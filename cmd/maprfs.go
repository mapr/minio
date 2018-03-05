package cmd

import (
	"io"
	"time"
	"runtime"
	"syscall"

	"github.com/minio/minio-go/pkg/policy"
	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
)

// MapRFSObjects - Wraps the FSObjects ObjectLayer implementation to add multitenancy support
type MapRFSObjects struct {
	*FSObjects
	fsUid int /// FS user id which should be used to access the file system
	fsGid int /// FS group id which should be used to access the file system
	prevUmask int /// Previous umask value, to be restored in ShutdownContext()
}

func (self *MapRFSObjects) PrepareContext() {
	runtime.LockOSThread()
	// TODO(RostakaGmfun): Implement setfsuid/setfsgid error handling
	syscall.Setfsuid(self.fsUid)
	syscall.Setfsgid(self.fsGid)
	self.prevUmask = syscall.Umask(0007) // TODO(RostakaGmfun): make umask configurable
}

func (self *MapRFSObjects) ShutdownContext() {
	syscall.Umask(self.prevUmask)
	syscall.Setfsuid(syscall.Geteuid())
	syscall.Setfsgid(syscall.Getegid())
	runtime.UnlockOSThread()
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
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.MakeBucketWithLocation(bucket, location)
}

func (self MapRFSObjects) GetBucketInfo(bucket string) (bucketInfo BucketInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.GetBucketInfo(bucket)
}

func (self MapRFSObjects) ListBuckets() (buckets []BucketInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListBuckets()
}

func (self MapRFSObjects) DeleteBucket(bucket string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.DeleteBucket(bucket)
}

func (self MapRFSObjects) ListObjects(bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListObjects(bucket, prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListObjectsV2(bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListObjectsV2(bucket, prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
}

func (self MapRFSObjects) GetObject(bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.GetObject(bucket, object, startOffset, length, writer, etag)
}

func (self MapRFSObjects) GetObjectInfo(bucket, object string) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.GetObjectInfo(bucket, object)
}

func (self MapRFSObjects) PutObject(bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.PutObject(bucket, object, data, metadata)
}

func (self MapRFSObjects) CopyObject(srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.CopyObject(srcBucket, srcObject, destBucket, destObject, srcInfo)
}

func (self MapRFSObjects) DeleteObject(bucket, object string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.DeleteObject(bucket, object)
}

func (self MapRFSObjects) ListMultipartUploads(bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListMultipartUploads(bucket, prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
}

func (self MapRFSObjects) NewMultipartUpload(bucket, object string, metadata map[string]string) (uploadID string, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.NewMultipartUpload(bucket, object, metadata)
}

func (self MapRFSObjects) CopyObjectPart(srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int, startOffset int64, length int64, srcInfo ObjectInfo) (info PartInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.CopyObjectPart(srcBucket, srcObject, destBucket, destObject, uploadID, partID, startOffset, length, srcInfo)
}

func (self MapRFSObjects) PutObjectPart(bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.PutObjectPart(bucket, object, uploadID, partID, data)
}

func (self MapRFSObjects) ListObjectParts(bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListObjectParts(bucket, object, uploadID, partNumberMarker, maxParts)
}

func (self MapRFSObjects) AbortMultipartUpload(bucket, object, uploadID string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.AbortMultipartUpload(bucket, object, uploadID)
}

func (self MapRFSObjects) CompleteMultipartUpload(bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.CompleteMultipartUpload(bucket, object, uploadID, uploadedParts)
}

func (self MapRFSObjects) HealFormat(dryRun bool) (madmin.HealResultItem, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.HealFormat(dryRun)
}

func (self MapRFSObjects) HealBucket(bucket string, dryRun bool) ([]madmin.HealResultItem, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.HealBucket(bucket, dryRun)
}

func (self MapRFSObjects) HealObject(bucket, object string, dryRun bool) (madmin.HealResultItem, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.HealObject(bucket, object, dryRun)
}

func (self MapRFSObjects) ListBucketsHeal() (buckets []BucketInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListBucketsHeal()
}

func (self MapRFSObjects) ListObjectsHeal(bucket, prefix, marker, delimiter string, maxKeys int) (ListObjectsInfo, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListObjectsHeal(bucket, prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListLocks(bucket, prefix string, duration time.Duration) ([]VolumeLockInfo, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ListLocks(bucket, prefix, duration)
}

func (self MapRFSObjects) ClearLocks(lockInfo []VolumeLockInfo) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.ClearLocks(lockInfo)
}

func (self MapRFSObjects) SetBucketPolicy(bucket string, policy policy.BucketAccessPolicy) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.SetBucketPolicy(bucket, policy)
}

func (self MapRFSObjects) GetBucketPolicy(bucket string) (policy.BucketAccessPolicy, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.GetBucketPolicy(bucket)
}

func (self MapRFSObjects) RefreshBucketPolicy(bucket string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.RefreshBucketPolicy(bucket)
}

func (self MapRFSObjects) DeleteBucketPolicy(bucket string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.DeleteBucketPolicy(bucket)
}

func (self MapRFSObjects) IsNotificationSupported() bool {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.IsNotificationSupported()
}

func (self MapRFSObjects) IsEncryptionSupported() bool {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.FSObjects.IsEncryptionSupported()
}
