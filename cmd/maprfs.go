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

// MapRFSObjects - Wraps the fsObjects ObjectLayer implementation to add multitenancy support
type MapRFSObjects struct {
	*fsObjects
	fsUid int /// FS user id which should be used to access the file system
	fsGid int /// FS group id which should be used to access the file system
}

func newMapRFSObjectLayer(path string) (ObjectLayer, error) {
	fsLayer, err := newFSObjectLayer(path)
	if err != nil {
		return nil, err
	}

	return &MapRFSObjects {
		fsObjects: fsLayer.(*fsObjects),
	}, nil
}

func (self *MapRFSObjects) PrepareContext() {
	runtime.LockOSThread()
	syscall.Setfsuid(self.fsUid)
	syscall.Setfsgid(self.fsGid)
	// TODO(RostakaGmfun): Change fsuid here
}

func (self MapRFSObjects) ShutdownContext() {
	// TODO(RostakaGmfun): Restore fsuid here
	runtime.UnlockOSThread()
}

func (self MapRFSObjects) Shutdown() error {
	return self.fsObjects.Shutdown()
}

func (self MapRFSObjects) StorageInfo() StorageInfo {
	var storageInfo = self.fsObjects.StorageInfo()
	storageInfo.Backend.Type = MapRFS
	return storageInfo
}

func (self MapRFSObjects) MakeBucketWithLocation(bucket, location string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.MakeBucketWithLocation(bucket, location)
}

func (self MapRFSObjects) GetBucketInfo(bucket string) (bucketInfo BucketInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.GetBucketInfo(bucket)
}

func (self MapRFSObjects) ListBuckets() (buckets []BucketInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListBuckets()
}

func (self MapRFSObjects) DeleteBucket(bucket string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.DeleteBucket(bucket)
}

func (self MapRFSObjects) ListObjects(bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListObjects(bucket, prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListObjectsV2(bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListObjectsV2(bucket, prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
}

func (self MapRFSObjects) GetObject(bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.GetObject(bucket, object, startOffset, length, writer, etag)
}

func (self MapRFSObjects) GetObjectInfo(bucket, object string) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.GetObjectInfo(bucket, object)
}

func (self MapRFSObjects) PutObject(bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.PutObject(bucket, object, data, metadata)
}

func (self MapRFSObjects) CopyObject(srcBucket, srcObject, destBucket, destObject string, metadata map[string]string, srcETag string) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.CopyObject(srcBucket, srcObject, destBucket, destObject, metadata, srcETag)
}

func (self MapRFSObjects) DeleteObject(bucket, object string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.DeleteObject(bucket, object)
}

func (self MapRFSObjects) ListMultipartUploads(bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListMultipartUploads(bucket, prefix, keyMarker, uploadIDMarker, delimiter, maxUploads)
}

func (self MapRFSObjects) NewMultipartUpload(bucket, object string, metadata map[string]string) (uploadID string, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.NewMultipartUpload(bucket, object, metadata)
}

func (self MapRFSObjects) CopyObjectPart(srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int, startOffset int64, length int64, metadata map[string]string, srcEtag string) (info PartInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.CopyObjectPart(srcBucket, srcObject, destBucket, destObject, uploadID, partID, startOffset, length, metadata, srcEtag)
}

func (self MapRFSObjects) PutObjectPart(bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.PutObjectPart(bucket, object, uploadID, partID, data)
}

func (self MapRFSObjects) ListObjectParts(bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListObjectParts(bucket, object, uploadID, partNumberMarker, maxParts)
}

func (self MapRFSObjects) AbortMultipartUpload(bucket, object, uploadID string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.AbortMultipartUpload(bucket, object, uploadID)
}

func (self MapRFSObjects) CompleteMultipartUpload(bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.CompleteMultipartUpload(bucket, object, uploadID, uploadedParts)
}

func (self MapRFSObjects) HealFormat(dryRun bool) (madmin.HealResultItem, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.HealFormat(dryRun)
}

func (self MapRFSObjects) HealBucket(bucket string, dryRun bool) ([]madmin.HealResultItem, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.HealBucket(bucket, dryRun)
}

func (self MapRFSObjects) HealObject(bucket, object string, dryRun bool) (madmin.HealResultItem, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.HealObject(bucket, object, dryRun)
}

func (self MapRFSObjects) ListBucketsHeal() (buckets []BucketInfo, err error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListBucketsHeal()
}

func (self MapRFSObjects) ListObjectsHeal(bucket, prefix, marker, delimiter string, maxKeys int) (ListObjectsInfo, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListObjectsHeal(bucket, prefix, marker, delimiter, maxKeys)
}

func (self MapRFSObjects) ListLocks(bucket, prefix string, duration time.Duration) ([]VolumeLockInfo, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ListLocks(bucket, prefix, duration)
}

func (self MapRFSObjects) ClearLocks(lockInfo []VolumeLockInfo) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.ClearLocks(lockInfo)
}

func (self MapRFSObjects) SetBucketPolicy(bucket string, policy policy.BucketAccessPolicy) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.SetBucketPolicy(bucket, policy)
}

func (self MapRFSObjects) GetBucketPolicy(bucket string) (policy.BucketAccessPolicy, error) {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.GetBucketPolicy(bucket)
}

func (self MapRFSObjects) RefreshBucketPolicy(bucket string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.RefreshBucketPolicy(bucket)
}

func (self MapRFSObjects) DeleteBucketPolicy(bucket string) error {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.DeleteBucketPolicy(bucket)
}

func (self MapRFSObjects) IsNotificationSupported() bool {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.IsNotificationSupported()
}

func (self MapRFSObjects) IsEncryptionSupported() bool {
	self.PrepareContext()
	defer self.ShutdownContext()
	return self.fsObjects.IsEncryptionSupported()
}
