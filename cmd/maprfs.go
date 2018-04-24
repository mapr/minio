package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"strings"
	"time"

	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio-go/pkg/policy"
)

// MapRFSObjects - Wraps the FSObjects ObjectLayer implementation to add multitenancy support
type MapRFSObjects struct {
	*FSObjects
	uid int /// FS user id which should be used to access the file system
	gid int /// FS group id which should be used to access the file system
	tenantName string /// Name of the tenant, used to evaluate bucket policies
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
	return statement.Actions.Contains(action)
}

func (self MapRFSObjects) getBucketOwner(bucket string) (uid int, gid int) {
	path := getBucketPath(bucket)
	fi, err := os.Stat(path)
	if err != nil {
		return self.uid, self.gid
	}

	linux_stat := fi.Sys().(*syscall.Stat_t)

	return int(linux_stat.Uid), int(linux_stat.Gid)
}

func (self MapRFSObjects) evaluateBucketPolicy(bucket, object string, policy policy.BucketAccessPolicy, action string) (uid int, gid int) {
	fmt.Println("eval bucket policy: ", policy)
	for _, statement := range policy.Statements {
		if statement.Effect == "Allow" &&
			matchPolicyTenant(self.tenantName, statement) &&
			matchPolicyAction(action, statement) &&
			matchPolicyResource(bucket, object, statement) {
				return self.getBucketOwner(bucket)
			}
	}

	return self.uid, self.gid
}

func (self MapRFSObjects) prepareContext(bucket, object, action string) error {
	policy := self.FSObjects.bucketPolicies.GetBucketPolicy(bucket)

	uid, gid := self.evaluateBucketPolicy(bucket, object, policy, action)

	runtime.LockOSThread()
	syscall.Setfsgid(gid)
	syscall.Setfsuid(uid)

	return nil
}

func (self MapRFSObjects) shutdownContext() error {
	syscall.Setfsuid(syscall.Geteuid())
	syscall.Setfsgid(syscall.Getegid())
	runtime.UnlockOSThread()
	return nil
}

/// Retrieve actual bucket name for the current tenant
func (self MapRFSObjects) getBucketName(bucket string) string {
	return bucket
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
	self.prepareContext(bucket, "", "s3:PutBucket")
	defer self.shutdownContext()
	return self.FSObjects.MakeBucketWithLocation(ctx, self.getBucketName(bucket), location)
}

func (self MapRFSObjects) GetBucketInfo(ctx context.Context, bucket string) (bucketInfo BucketInfo, err error) {
	self.prepareContext(bucket, "", "s3:GetBucketInfo")
	defer self.shutdownContext()
	return self.FSObjects.GetBucketInfo(ctx, self.getBucketName(bucket))
}

func (self MapRFSObjects) ListBuckets(ctx context.Context) (buckets []BucketInfo, err error) {
	// No need to perform impersonation as all buckets are visiblae to all tenants
	return self.FSObjects.ListBuckets(ctx)
}

func (self MapRFSObjects) DeleteBucket(ctx context.Context, bucket string) error {
	self.prepareContext(bucket, "", "s3:DeleteBucket")
	defer self.shutdownContext()
	return self.FSObjects.DeleteBucket(ctx, self.getBucketName(bucket))
}

func (self MapRFSObjects) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	self.prepareContext(bucket, "", "s3:ListBucket")
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
	return self.FSObjects.ListObjects(ctx, self.getBucketName(bucket), prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	self.prepareContext(bucket, "", "s3:ListObjects")
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
	return self.FSObjects.ListObjectsV2(ctx, self.getBucketName(bucket), prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
}

func (self MapRFSObjects) GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (error) {
	err := self.prepareContext(bucket, object, "s3:GetObject")
	defer self.shutdownContext()
	if err != nil {
		return err
	}
	return self.FSObjects.GetObject(ctx, self.getBucketName(bucket), object, startOffset, length, writer, etag)
}

func (self MapRFSObjects) GetObjectInfo(ctx context.Context, bucket, object string) (objInfo ObjectInfo, err error) {
	self.prepareContext(bucket, object, "s3:GetObject")
	defer self.shutdownContext()
	return self.FSObjects.GetObjectInfo(ctx, self.getBucketName(bucket), object)
}

func (self MapRFSObjects) PutObject(ctx context.Context, bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error) {
	self.prepareContext(bucket, object, "s3:PutObject")
	defer self.shutdownContext()
	return self.FSObjects.PutObject(ctx, self.getBucketName(bucket), object, data, metadata)
}

func (self MapRFSObjects) CopyObject(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo) (objInfo ObjectInfo, err error) {
	self.prepareContext("", "", "s3:CopyObject")
	defer self.shutdownContext()
	return self.FSObjects.CopyObject(ctx, srcBucket, srcObject, destBucket, destObject, srcInfo)
}

func (self MapRFSObjects) DeleteObject(ctx context.Context, bucket, object string) error {
	self.prepareContext(bucket, object, "s3:DeleteObject")
	defer self.shutdownContext()
	return self.FSObjects.DeleteObject(ctx, self.getBucketName(bucket), object)
}

func (self MapRFSObjects) ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	self.prepareContext(bucket, "", "s3:ListBucketMultipartUploads")
	defer self.shutdownContext()
	return self.FSObjects.ListMultipartUploads(ctx, self.getBucketName(bucket), prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
}

func (self MapRFSObjects) NewMultipartUpload(ctx context.Context, bucket, object string, metadata map[string]string) (uploadID string, err error) {
	self.prepareContext(bucket, object, "s3:PutObject")
	defer self.shutdownContext()
	return self.FSObjects.NewMultipartUpload(ctx, self.getBucketName(bucket), object, metadata)
}

func (self MapRFSObjects) CopyObjectPart(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int, startOffset int64, length int64, srcInfo ObjectInfo) (info PartInfo, err error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.CopyObjectPart(ctx, srcBucket, srcObject, destBucket, destObject, uploadID, partID, startOffset, length, srcInfo)
}

func (self MapRFSObjects) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error) {
	self.prepareContext(bucket, object, "")
	defer self.shutdownContext()
	return self.FSObjects.PutObjectPart(ctx, self.getBucketName(bucket), object, uploadID, partID, data)
}

func (self MapRFSObjects) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error) {
	self.prepareContext(bucket,  object, "s3:ListObjectParts")
	defer self.shutdownContext()
	return self.FSObjects.ListObjectParts(ctx, self.getBucketName(bucket), object, uploadID, partNumberMarker, maxParts)
}

func (self MapRFSObjects) AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.AbortMultipartUpload(ctx, self.getBucketName(bucket), object, uploadID)
}

func (self MapRFSObjects) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.CompleteMultipartUpload(ctx, self.getBucketName(bucket), object, uploadID, uploadedParts)
}

func (self MapRFSObjects) HealFormat(ctx context.Context, dryRun bool) (madmin.HealResultItem, error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.HealFormat(ctx, dryRun)
}

func (self MapRFSObjects) HealBucket(ctx context.Context, bucket string, dryRun bool) ([]madmin.HealResultItem, error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.HealBucket(ctx, self.getBucketName(bucket), dryRun)
}

func (self MapRFSObjects) HealObject(ctx context.Context, bucket, object string, dryRun bool) (madmin.HealResultItem, error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.HealObject(ctx, self.getBucketName(bucket), object, dryRun)
}

func (self MapRFSObjects) ListBucketsHeal(ctx context.Context) (buckets []BucketInfo, err error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.ListBucketsHeal(ctx)
}

func (self MapRFSObjects) ListObjectsHeal(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (ListObjectsInfo, error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.ListObjectsHeal(ctx, self.getBucketName(bucket), prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListLocks(ctx context.Context, bucket, prefix string, duration time.Duration) ([]VolumeLockInfo, error) {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.ListLocks(ctx, self.getBucketName(bucket), prefix, duration)
}

func (self MapRFSObjects) ClearLocks(ctx context.Context, lockInfo []VolumeLockInfo) error {
	self.prepareContext("", "", "")
	defer self.shutdownContext()
	return self.FSObjects.ClearLocks(ctx, lockInfo)
}

func (self MapRFSObjects) SetBucketPolicy(ctx context.Context, bucket string, policy policy.BucketAccessPolicy) error {
	self.prepareContext(bucket, "", "s3:PutBucketPolicy")
	defer self.shutdownContext()
	return self.FSObjects.SetBucketPolicy(ctx, self.getBucketName(bucket), policy)
	if err != nil {
		return err
	}
}

func (self MapRFSObjects) GetBucketPolicy(ctx context.Context, bucket string) (policy.BucketAccessPolicy, error) {
	self.prepareContext(bucket, "", "s3:GetBucketPolicy")
	defer self.shutdownContext()
	return self.FSObjects.GetBucketPolicy(ctx, self.getBucketName(bucket))
}

func (self MapRFSObjects) RefreshBucketPolicy(ctx context.Context, bucket string) error {
	self.prepareContext(bucket, "", "")
	defer self.shutdownContext()
	return self.FSObjects.RefreshBucketPolicy(ctx, self.getBucketName(bucket))
}

func (self MapRFSObjects) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	self.prepareContext(bucket, "", "s3:DeleteBucketPolicy")
	defer self.shutdownContext()
	err := self.FSObjects.DeleteBucketPolicy(ctx, self.getBucketName(bucket))
	if err != nil {
		return err
	}

	if self.withMaprAce {
		return RemoveMapRFSBucketPolicy(bucket, self.FSObjects.bucketPolicies.GetBucketPolicy(bucket))
	} else {
		return err
	}
}

func (self MapRFSObjects) IsNotificationSupported() bool {
	return self.FSObjects.IsNotificationSupported()
}

func (self MapRFSObjects) IsEncryptionSupported() bool {
	return self.FSObjects.IsEncryptionSupported()
}
