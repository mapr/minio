package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"syscall"
	"strings"
	"time"

	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio-go/pkg/policy"
	"github.com/minio/minio-go/pkg/set"
)

// MapRFSObjects - Wraps the FSObjects ObjectLayer implementation to add multitenancy support
type MapRFSObjects struct {
	*FSObjects
	uid int /// FS user id which should be used to access the file system
	gid int /// FS group id which should be used to access the file system
	tenantName string /// Name of the tenant, used to evaluate bucket policies
	securityScenario string /// Security scenariospecified at server start
}

func matchPolicyResource(bucket, object string, statement policy.Statement) bool {
	resourceString := "arn:aws:s3:::" + bucket + "/" + object
	resourceString = strings.TrimSuffix(resourceString, "/")
	return statement.Resources.Contains(resourceString) ||
		statement.Resources.Contains("arn:aws:s3:::" + bucket + "/*")
}

func matchPolicyTenant(tenantName string, statement policy.Statement) bool {
	return statement.Principal.AWS.Contains(tenantName) || statement.Principal.AWS.Contains("*")
}

func matchPolicyAction(action string, statement policy.Statement) bool {
	return statement.Actions.Contains(action) || statement.Actions.Contains("s3:*")
}

var writableActions = set.StringSet{
	"s3:PutObject": {},
	"s3:DeleteObject": {},
}

/// Returns true if a given action requires write permission
func actionIsWritable(action string) bool {
	_, ok := writableActions[action];
	return ok
}

const defaultBucketPolicyJson = `
{
   "Version": "version",
   "Statement": [
       {
           "Effect": "Allow",
           "Principal": { "AWS": ["insert-tenant-name-here"] },
           "Action": ["s3:*"],
           "Resource": "arn:aws:s3:::insert-resource-here",
           "Sid": "1"
       }
   ]
}
`

func getBucketPath(bucket string) string {
	objectApi := newObjectLayerFn(nil)
	return pathJoin(objectApi.(*FSObjects).fsPath, bucket)
}

func getObjectPath(bucket, object string) string {
	return path.Join(getBucketPath(bucket), object)
}

func getObjectMetaPath(bucket, object string) string {
	objectApi := newObjectLayerFn(nil)
	return pathJoin(objectApi.(*FSObjects).fsPath, ".minio.sys", "buckets", bucket, object)
}

func getBucketOwner(bucket string) (err error, uid int, gid int) {
	path := getBucketPath(bucket)
	fi, err := os.Stat(path)
	if err != nil {
		return err, 0, 0
	}

	linux_stat := fi.Sys().(*syscall.Stat_t)

	return nil, int(linux_stat.Uid), int(linux_stat.Gid)
}

func getObjectOwner(bucket, object string) (err error, uid int, gid int) {
	path := getObjectPath(bucket, object)
	fi, err := os.Stat(path)
	if err != nil {
		return err, 0, 0
	}

	linux_stat := fi.Sys().(*syscall.Stat_t)

	return nil, int(linux_stat.Uid), int(linux_stat.Gid)
}

func (self MapRFSObjects) matchBucketPolicy(bucket, object string, policy policy.BucketAccessPolicy, action string) bool {
	for _, statement := range policy.Statements {
		if statement.Effect == "Allow" &&
			matchPolicyTenant(self.tenantName, statement) &&
			matchPolicyAction(action, statement) &&
			matchPolicyResource(bucket, object, statement) {
			return true
		}
	}
	return false
}

func (self MapRFSObjects) evaluateUidGid(bucket, object, action string) (error, int, int) {
	err, bucketUid, bucketGid := getBucketOwner(bucket)
	fmt.Println("err ", err, bucketUid, bucketGid)
	if err != nil {
		return err, self.uid, self.gid
	}

	if object == "" || !actionIsWritable(action) || (self.uid == bucketUid && self.gid == bucketGid) {
		return nil, bucketUid, bucketGid
	}

	err, uid, gid := getObjectOwner(bucket, object)
	fmt.Println("obj owner ", err, uid, gid)
	if err != nil || (uid == self.uid && gid == self.gid) {
		return nil, bucketUid, bucketGid
	} else {
		return PrefixAccessDenied{bucket, object}, 0, 0
	}
}

func (self MapRFSObjects) evaluateBucketPolicy(bucket, object string, policy policy.BucketAccessPolicy, action string) (err error, uid int, gid int) {
	fmt.Println("eval bucket policy: ", policy, bucket, object, action)
	err, bucketUid, bucketGid := getBucketOwner(bucket)
	if self.matchBucketPolicy(bucket, object, policy, action) || (self.uid == bucketUid && self.gid == bucketGid) {
		return self.evaluateUidGid(bucket, object, action)
	}

	return PrefixAccessDenied{bucket, object}, 0, 0
}

func (self MapRFSObjects) prepareContextHybrid(bucket, object, action string) error {
	policy := self.FSObjects.bucketPolicies.GetBucketPolicy(bucket)

	err, uid, gid := self.evaluateBucketPolicy(bucket, object, policy, action)
	if err != nil {
		return err
	}

	runtime.LockOSThread()
	syscall.Setfsgid(gid)
	syscall.Setfsuid(uid)

	return nil
}

func (self MapRFSObjects) prepareContextFSOnly(bucket, object, action string) error {
	runtime.LockOSThread()
	syscall.Setfsgid(self.gid)
	syscall.Setfsuid(self.uid)

	return nil
}

func (self MapRFSObjects) prepareContextS3Only(bucket, object, action string) error {
	policy := self.FSObjects.bucketPolicies.GetBucketPolicy(bucket)
	fmt.Println("eval policy:", bucket, object, action, self.tenantName)
	if self.matchBucketPolicy(bucket, object, policy, action) {
		return nil
	} else {
		return PrefixAccessDenied{bucket, object}
	}
}

func (self MapRFSObjects) prepareContext(bucket, object, action string) error {
	switch self.securityScenario {
	case "hybrid":
		return self.prepareContextHybrid(bucket, object, action)
	case "fs_only":
		return self.prepareContextFSOnly(bucket, object, action)
	case "s3_only":
		return self.prepareContextS3Only(bucket, object, action)
	default:
		return errInvalidArgument
	}
}

func (self MapRFSObjects) shutdownContextHybrid() error {
	syscall.Setfsuid(syscall.Geteuid())
	syscall.Setfsgid(syscall.Getegid())
	runtime.UnlockOSThread()
	return nil
}

func (self MapRFSObjects) shutdownContextFSOnly() error {
	syscall.Setfsuid(syscall.Geteuid())
	syscall.Setfsgid(syscall.Getegid())
	runtime.UnlockOSThread()
	return nil
}

func (self MapRFSObjects) shutdownContextS3Only() error {
	return nil
}

func (self MapRFSObjects) shutdownContext() error {
	switch self.securityScenario {
	case "hybrid":
		return self.shutdownContextHybrid()
	case "fs_only":
		return self.shutdownContextFSOnly()
	case "s3":
		return self.shutdownContextS3Only()
	default:
		return errInvalidArgument
	}
}

func (self MapRFSObjects) Shutdown(ctx context.Context) error {
	return self.FSObjects.Shutdown(ctx)
}

func (self MapRFSObjects) StorageInfo(ctx context.Context) StorageInfo {
	var storageInfo = self.FSObjects.StorageInfo(ctx)
	storageInfo.Backend.Type = MapRFS
	return storageInfo
}

func (self MapRFSObjects) MakeBucketWithLocation(ctx context.Context, bucket, location string) error {
	// Create bucket directory using the current 'root' user and then chown
	err := self.FSObjects.MakeBucketWithLocation(ctx, bucket, location)
	if err != nil {
		return err
	}

	if self.securityScenario == "s3_only" {
		var bucketPolicy policy.BucketAccessPolicy
		err = parseBucketPolicy(strings.NewReader(defaultBucketPolicyJson), &bucketPolicy)
		if err != nil {
			fmt.Println("parse bucket policy err", err)
			return err
		}
		bucketPolicy.Statements[0].Principal.AWS = set.CreateStringSet(self.tenantName)
		bucketPolicy.Statements[0].Resources = set.CreateStringSet("arn:aws:s3:::" + bucket)
		fmt.Println("Assigning default bucket policy:", bucketPolicy)
		err := self.FSObjects.SetBucketPolicy(ctx, bucket, bucketPolicy)
		if err != nil {
			fmt.Println("set bucket policy err", err)
			return err
		}
	}

	err = os.Chown(getBucketPath(bucket), self.uid, self.gid);
	if err != nil {
		return err
	}

	bucketMetaDir := pathJoin(self.FSObjects.fsPath, minioMetaBucket, bucketMetaPrefix, bucket)
	return os.Chown(bucketMetaDir, self.uid, self.gid);
}

func (self MapRFSObjects) GetBucketInfo(ctx context.Context, bucket string) (bucketInfo BucketInfo, err error) {
	if err = self.prepareContext(bucket, "", "s3:GetBucketInfo"); err != nil {
		return bucketInfo, err
	}
	defer self.shutdownContext()
	return self.FSObjects.GetBucketInfo(ctx, bucket)
}

func (self MapRFSObjects) ListBuckets(ctx context.Context) (buckets []BucketInfo, err error) {
	// No need to perform impersonation as all buckets are visiblae to all tenants
	return self.FSObjects.ListBuckets(ctx)
}

func (self MapRFSObjects) DeleteBucket(ctx context.Context, bucket string) error {
	policy := self.FSObjects.bucketPolicies.GetBucketPolicy(bucket)

	_, uid, gid := getBucketOwner(bucket)

	if !self.matchBucketPolicy(bucket, "", policy, "s3:DeleteBucket") &&
		(uid != self.uid || gid != self.gid) {
		return PrefixAccessDenied{}
	}

	// Bypass fs impersonation since only user who created directory can delete it
	return self.FSObjects.DeleteBucket(ctx, bucket)
}

func (self MapRFSObjects) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	if err := self.prepareContext(bucket, "", "s3:ListBucket"); err != nil {
		return result, err
	}

	defer self.shutdownContext()

	// Temporary hack to handle access denied for ListObjects,
	// since tree walk in fs-v1 is done in the context of another thread.
	// TODO(RostakaGmfun): either rewrite fs-v1.ListObjects
	// or update treeWalk to use fs impersonation.
	f, err := os.Open(getBucketPath(bucket))
	if err != nil {
		return result, PrefixAccessDenied{}
	}
	f.Close()
	return self.FSObjects.ListObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	if err := self.prepareContext(bucket, "", "s3:ListBucket"); err != nil {
		return result, err
	}

	defer self.shutdownContext()

	// Temporary hack to handle access denied for ListObjects,
	// since tree walk in fs-v1 is done in the context of another thread.
	// TODO(RostakaGmfun): either rewrite fs-v1.ListObjects
	// or update treeWalk to use fs impersonation.
	f, err := os.Open(getBucketPath(bucket))
	if err != nil {
		return result, PrefixAccessDenied{}
	}
	f.Close()
	return self.FSObjects.ListObjectsV2(ctx, bucket, prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
}

func (self MapRFSObjects) GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (error) {
	if err := self.prepareContext(bucket, object, "s3:GetObject"); err != nil {
		return err
	}
	defer self.shutdownContext()
	return self.FSObjects.GetObject(ctx, bucket, object, startOffset, length, writer, etag)
}

func (self MapRFSObjects) GetObjectInfo(ctx context.Context, bucket, object string) (objInfo ObjectInfo, err error) {
	if err := self.prepareContext(bucket, object, "s3:GetObject"); err != nil {
		return objInfo, err
	}
	defer self.shutdownContext()
	return self.FSObjects.GetObjectInfo(ctx, bucket, object)
}

func (self MapRFSObjects) PutObject(ctx context.Context, bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error) {
	if err := self.prepareContext(bucket, object, "s3:PutObject"); err != nil {
		fmt.Println("failed to put object", err)
		return objInfo, err
	}
	objInfo, err = self.FSObjects.PutObject(ctx, bucket, object, data, metadata)
	self.shutdownContext()
	if err != nil {
		return objInfo, err
	}

	ret := os.Chown(getObjectPath(bucket, object), self.uid, self.gid);
	if ret != nil {
		return ObjectInfo{}, ret
	}

	if ret != nil {
		return ObjectInfo{}, ret
	}

	return objInfo, err
}

func (self MapRFSObjects) CopyObject(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo) (objInfo ObjectInfo, err error) {
	if err := self.prepareContext("", "", "s3:CopyObject"); err != nil {
		return objInfo, err
	}
	defer self.shutdownContext()
	return self.FSObjects.CopyObject(ctx, srcBucket, srcObject, destBucket, destObject, srcInfo)
}

func (self MapRFSObjects) DeleteObject(ctx context.Context, bucket, object string) error {
	if err := self.prepareContext(bucket, object, "s3:DeleteObject"); err != nil {
		return err
	}
	defer self.shutdownContext()
	return self.FSObjects.DeleteObject(ctx, bucket, object)
}

func (self MapRFSObjects) ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	if err := self.prepareContext(bucket, "", "s3:ListBucketMultipartUploads"); err != nil {
		return result, err
	}
	defer self.shutdownContext()
	return self.FSObjects.ListMultipartUploads(ctx, bucket, prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
}

func (self MapRFSObjects) NewMultipartUpload(ctx context.Context, bucket, object string, metadata map[string]string) (uploadID string, err error) {
	if err := self.prepareContext(bucket, object, "s3:PutObject"); err != nil {
		return uploadID, err
	}
	defer self.shutdownContext()
	return self.FSObjects.NewMultipartUpload(ctx, bucket, object, metadata)
}

func (self MapRFSObjects) CopyObjectPart(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int, startOffset int64, length int64, srcInfo ObjectInfo) (info PartInfo, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return info, err
	}
	defer self.shutdownContext()
	return self.FSObjects.CopyObjectPart(ctx, srcBucket, srcObject, destBucket, destObject, uploadID, partID, startOffset, length, srcInfo)
}

func (self MapRFSObjects) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error) {
	if err := self.prepareContext(bucket, object, ""); err != nil {
		return info, err
	}
	defer self.shutdownContext()
	return self.FSObjects.PutObjectPart(ctx, bucket, object, uploadID, partID, data)
}

func (self MapRFSObjects) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error) {
	if err := self.prepareContext(bucket,  object, "s3:ListObjectParts"); err != nil {
		return result, err
	}
	defer self.shutdownContext()
	return self.FSObjects.ListObjectParts(ctx, bucket, object, uploadID, partNumberMarker, maxParts)
}

func (self MapRFSObjects) AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error {
	if err := self.prepareContext("", "", ""); err != nil {
		return err
	}
	defer self.shutdownContext()
	return self.FSObjects.AbortMultipartUpload(ctx, bucket, object, uploadID)
}

func (self MapRFSObjects) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return objInfo, err
	}
	defer self.shutdownContext()
	return self.FSObjects.CompleteMultipartUpload(ctx, bucket, object, uploadID, uploadedParts)
}

func (self MapRFSObjects) HealFormat(ctx context.Context, dryRun bool) (result madmin.HealResultItem, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return result, err
	}
	defer self.shutdownContext()
	return self.FSObjects.HealFormat(ctx, dryRun)
}

func (self MapRFSObjects) HealBucket(ctx context.Context, bucket string, dryRun bool) (result []madmin.HealResultItem, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return result, err
	}
	defer self.shutdownContext()
	return self.FSObjects.HealBucket(ctx, bucket, dryRun)
}

func (self MapRFSObjects) HealObject(ctx context.Context, bucket, object string, dryRun bool) (result madmin.HealResultItem, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return result, err
	}
	defer self.shutdownContext()
	return self.FSObjects.HealObject(ctx, bucket, object, dryRun)
}

func (self MapRFSObjects) ListBucketsHeal(ctx context.Context) (buckets []BucketInfo, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return buckets, err
	}
	defer self.shutdownContext()
	return self.FSObjects.ListBucketsHeal(ctx)
}

func (self MapRFSObjects) ListObjectsHeal(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (info ListObjectsInfo, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return info, err
	}
	defer self.shutdownContext()
	return self.FSObjects.ListObjectsHeal(ctx, bucket, prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListLocks(ctx context.Context, bucket, prefix string, duration time.Duration) (info []VolumeLockInfo, err error) {
	if err := self.prepareContext("", "", ""); err != nil {
		return info, err
	}
	defer self.shutdownContext()
	return self.FSObjects.ListLocks(ctx, bucket, prefix, duration)
}

func (self MapRFSObjects) ClearLocks(ctx context.Context, lockInfo []VolumeLockInfo) error {
	if err := self.prepareContext("", "", ""); err != nil {
		return err
	}
	defer self.shutdownContext()
	return self.FSObjects.ClearLocks(ctx, lockInfo)
}

func (self MapRFSObjects) SetBucketPolicy(ctx context.Context, bucket string, policy policy.BucketAccessPolicy) error {
	if err := self.prepareContext(bucket, "", "s3:PutBucketPolicy"); err != nil {
		return err
	}
	defer self.shutdownContext()

	err, uid, gid := getBucketOwner(bucket)
	if err != nil || uid != self.uid || gid != self.gid {
		return PrefixAccessDenied{}
	}
	return self.FSObjects.SetBucketPolicy(ctx, bucket, policy)
}

func (self MapRFSObjects) GetBucketPolicy(ctx context.Context, bucket string) (policy policy.BucketAccessPolicy, err error) {
	if err := self.prepareContext(bucket, "", "s3:GetBucketPolicy"); err != nil {
		return policy, err
	}
	defer self.shutdownContext()

	err, uid, gid := getBucketOwner(bucket)
	if err != nil || uid != self.uid || gid != self.gid {
		return policy, PrefixAccessDenied{}
	}
	return self.FSObjects.GetBucketPolicy(ctx, bucket)
}

func (self MapRFSObjects) RefreshBucketPolicy(ctx context.Context, bucket string) error {
	if err := self.prepareContext(bucket, "", ""); err != nil {
		return err
	}
	defer self.shutdownContext()
	return self.FSObjects.RefreshBucketPolicy(ctx, bucket)
}

func (self MapRFSObjects) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	if err := self.prepareContext(bucket, "", "s3:DeleteBucketPolicy"); err != nil {
		return err
	}
	defer self.shutdownContext()
	return self.FSObjects.DeleteBucketPolicy(ctx, bucket)
}

func (self MapRFSObjects) IsNotificationSupported() bool {
	return self.FSObjects.IsNotificationSupported()
}

func (self MapRFSObjects) IsEncryptionSupported() bool {
	return self.FSObjects.IsEncryptionSupported()
}
