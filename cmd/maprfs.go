package cmd

import (
	"io"
	"time"
	"runtime"
	"syscall"
	"strings"

	"github.com/minio/minio-go/pkg/policy"
	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
)

// MapRFSObjects - Wraps the FSObjects ObjectLayer implementation to add multitenancy support
type MapRFSObjects struct {
	*FSObjects
	fsUid int /// FS user id which should be used to access the file system
	fsGid int /// FS group id which should be used to access the file system
	prevUmask int /// Previous umask value, to be restored in phutdownContext()
	tenantPrefix string /// Used to prefix buckets in order to allow multiple tenants have buckets with the same name
}

func (self MapRFSObjects) prepareContext() {
	runtime.LockOSThread()
	// TODO(RostakaGmfun): Implement setfsuid/setfsgid error handling
	syscall.Setfsuid(self.fsUid)
	syscall.Setfsgid(self.fsGid)
	self.prevUmask = syscall.Umask(0007) // TODO(RostakaGmfun): make umask configurable
}

func (self MapRFSObjects) shutdownContext() {
	syscall.Umask(self.prevUmask)
	syscall.Setfsuid(syscall.Geteuid())
	syscall.Setfsgid(syscall.Getegid())
	runtime.UnlockOSThread()
}

/// Retrieve actual bucket name for the current tenant
func (self MapRFSObjects) getBucketName(bucket string) string {
	// TODO(RostakaGmfun): Validate tenantName somewhere
	return self.tenantPrefix + bucket
}

func (self MapRFSObjects) Shutdown() error {
	return self.FSObjects.Shutdown()
}

func (self MapRFSObjects) StorageInfo() StorageInfo {
	var storageInfo = self.FSObjects.StorageInfo()
	storageInfo.Backend.Type = MapRFS
	return storageInfo
}

func (self MapRFSObjects) MakeBucketWithLocation(bucket, location string) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.MakeBucketWithLocation(self.getBucketName(bucket), location)
}

func (self MapRFSObjects) GetBucketInfo(bucket string) (bucketInfo BucketInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.GetBucketInfo(self.getBucketName(bucket))
}

func (self MapRFSObjects) ListBuckets() (buckets []BucketInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	allBuckets, err := self.FSObjects.ListBuckets()

	var visibleBuckets []BucketInfo

	/// Filter out buckets which don't belong to the current tenant
	/// and fixup bucket names by removing tenantPrefix
	for _, bucket := range allBuckets {
		if strings.HasPrefix(bucket.Name, self.tenantPrefix) {
			visibleBuckets = append(visibleBuckets, BucketInfo{
				Name: strings.TrimPrefix(bucket.Name, self.tenantPrefix),
				Created: bucket.Created,
			})
		}
	}

	return visibleBuckets, err
}

func (self MapRFSObjects) DeleteBucket(bucket string) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.DeleteBucket(self.getBucketName(bucket))
}

func (self MapRFSObjects) ListObjects(bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListObjects(self.getBucketName(bucket), prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListObjectsV2(bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListObjectsV2(self.getBucketName(bucket), prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
}

func (self MapRFSObjects) GetObject(bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.GetObject(self.getBucketName(bucket), object, startOffset, length, writer, etag)
}

func (self MapRFSObjects) GetObjectInfo(bucket, object string) (objInfo ObjectInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.GetObjectInfo(self.getBucketName(bucket), object)
}

func (self MapRFSObjects) PutObject(bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.PutObject(self.getBucketName(bucket), object, data, metadata)
}

func (self MapRFSObjects) CopyObject(srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo) (objInfo ObjectInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.CopyObject(srcBucket, srcObject, destBucket, destObject, srcInfo)
}

func (self MapRFSObjects) DeleteObject(bucket, object string) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.DeleteObject(self.getBucketName(bucket), object)
}

func (self MapRFSObjects) ListMultipartUploads(bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListMultipartUploads(self.getBucketName(bucket), prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
}

func (self MapRFSObjects) NewMultipartUpload(bucket, object string, metadata map[string]string) (uploadID string, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.NewMultipartUpload(self.getBucketName(bucket), object, metadata)
}

func (self MapRFSObjects) CopyObjectPart(srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int, startOffset int64, length int64, srcInfo ObjectInfo) (info PartInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.CopyObjectPart(srcBucket, srcObject, destBucket, destObject, uploadID, partID, startOffset, length, srcInfo)
}

func (self MapRFSObjects) PutObjectPart(bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.PutObjectPart(self.getBucketName(bucket), object, uploadID, partID, data)
}

func (self MapRFSObjects) ListObjectParts(bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListObjectParts(self.getBucketName(bucket), object, uploadID, partNumberMarker, maxParts)
}

func (self MapRFSObjects) AbortMultipartUpload(bucket, object, uploadID string) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.AbortMultipartUpload(self.getBucketName(bucket), object, uploadID)
}

func (self MapRFSObjects) CompleteMultipartUpload(bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.CompleteMultipartUpload(self.getBucketName(bucket), object, uploadID, uploadedParts)
}

func (self MapRFSObjects) HealFormat(dryRun bool) (madmin.HealResultItem, error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.HealFormat(dryRun)
}

func (self MapRFSObjects) HealBucket(bucket string, dryRun bool) ([]madmin.HealResultItem, error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.HealBucket(self.getBucketName(bucket), dryRun)
}

func (self MapRFSObjects) HealObject(bucket, object string, dryRun bool) (madmin.HealResultItem, error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.HealObject(self.getBucketName(bucket), object, dryRun)
}

func (self MapRFSObjects) ListBucketsHeal() (buckets []BucketInfo, err error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListBucketsHeal()
}

func (self MapRFSObjects) ListObjectsHeal(bucket, prefix, marker, delimiter string, maxKeys int) (ListObjectsInfo, error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListObjectsHeal(self.getBucketName(bucket), prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListLocks(bucket, prefix string, duration time.Duration) ([]VolumeLockInfo, error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ListLocks(self.getBucketName(bucket), prefix, duration)
}

func (self MapRFSObjects) ClearLocks(lockInfo []VolumeLockInfo) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.ClearLocks(lockInfo)
}

func (self MapRFSObjects) SetBucketPolicy(bucket string, policy policy.BucketAccessPolicy) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.SetBucketPolicy(self.getBucketName(bucket), policy)
}

func (self MapRFSObjects) GetBucketPolicy(bucket string) (policy.BucketAccessPolicy, error) {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.GetBucketPolicy(self.getBucketName(bucket))
}

func (self MapRFSObjects) RefreshBucketPolicy(bucket string) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.RefreshBucketPolicy(self.getBucketName(bucket))
}

func (self MapRFSObjects) DeleteBucketPolicy(bucket string) error {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.DeleteBucketPolicy(self.getBucketName(bucket))
}

func (self MapRFSObjects) IsNotificationSupported() bool {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.IsNotificationSupported()
}

func (self MapRFSObjects) IsEncryptionSupported() bool {
	self.prepareContext()
	defer self.shutdownContext()
	return self.FSObjects.IsEncryptionSupported()
}
