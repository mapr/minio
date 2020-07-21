package cmd

import (
	"context"
	"github.com/minio/minio/cmd/logger"
	"net/http"
	"os"
	"path"
	"strconv"
	"syscall"
)

type MapRFSObjects struct {
	*FSObjects
}

// NewMapRFSObjectLayer - initialize new fs object layer.
func NewMapRFSObjectLayer(fsPath string) (ObjectLayer, error) {
	fs, err := NewFSObjectLayer(fsPath)

	if err != nil {
		return nil, err
	}

	logger.Info("Started with UID: " + strconv.Itoa(syscall.Geteuid()) + " GID: " + strconv.Itoa(syscall.Getgid()))

	return &MapRFSObjects{
		FSObjects: fs.(*FSObjects),
	}, err
}

func (fs MapRFSObjects) MakeBucketWithLocation(ctx context.Context, bucket string, opts BucketOptions) error {
	if err := PrepareContext(ctx); err != nil {
		return err
	}
	defer ShutdownContext()

	err := fs.FSObjects.MakeBucketWithLocation(ctx, bucket, opts)
	if err == errDiskAccessDenied {
		return errAccessDenied
	}

	return err
}

func (fs MapRFSObjects) DeleteBucket(ctx context.Context, bucket string, forceDelete bool) error {
	if err := PrepareContext(ctx); err != nil {
		return err
	}
	defer ShutdownContext()

	err := fs.FSObjects.DeleteBucket(ctx, bucket, forceDelete)
	if err != nil && os.IsPermission(err) {
		return PrefixAccessDenied{
			Bucket: bucket,
			Object: "/",
		}
	}

	return err
}

func (fs MapRFSObjects) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	if err := PrepareContext(ctx); err != nil {
		return ListObjectsInfo{}, err
	}
	defer ShutdownContext()

	// Temporary hack to handle access denied for ListObjects,
	// since tree walk in fs-v1 is done in the context of another thread.
	// TODO: either rewrite fs-v1.ListObjects
	// or update treeWalk to use fs impersonation.
	if err := fs.checkReadListPermissions(ctx, bucket, prefix, delimiter); err != nil {
		return result, err
	}

	return fs.FSObjects.ListObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
}

func (fs MapRFSObjects) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	if err := PrepareContext(ctx); err != nil {
		return ListObjectsV2Info{}, err
	}
	defer ShutdownContext()

	// Temporary hack to handle access denied for ListObjects,
	// since tree walk in fs-v1 is done in the context of another thread.
	// TODO: either rewrite fs-v1.ListObjects
	// or update treeWalk to use fs impersonation.
	if err := fs.checkReadListPermissions(ctx, bucket, prefix, delimiter); err != nil {
		return result, err
	}

	return fs.FSObjects.ListObjectsV2(ctx, bucket, prefix, continuationToken, delimiter, maxKeys, fetchOwner, startAfter)
}

func (fs MapRFSObjects) GetObjectNInfo(ctx context.Context, bucket, object string, rs *HTTPRangeSpec, h http.Header, lockType LockType, opts ObjectOptions) (reader *GetObjectReader, err error) {
	if err := PrepareContext(ctx); err != nil {
		return nil, err
	}
	defer ShutdownContext()

	return fs.FSObjects.GetObjectNInfo(ctx, bucket, object, rs, h, lockType, opts)
}

func (fs MapRFSObjects) GetObjectInfo(ctx context.Context, bucket, object string, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	if err := PrepareContext(ctx); err != nil {
		return ObjectInfo{}, err
	}
	defer ShutdownContext()

	return fs.FSObjects.GetObjectInfo(ctx, bucket, object, opts)
}

func (fs MapRFSObjects) DeleteObject(ctx context.Context, bucket, object string, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	defer ShutdownContext()

	return fs.FSObjects.DeleteObject(ctx, bucket, object, opts)
}

func (fs MapRFSObjects) DeleteObjects(ctx context.Context, bucket string, objects []ObjectToDelete, opts ObjectOptions) ([]DeletedObject, []error) {
	if err := PrepareContext(ctx); err != nil {
		errs := make([]error, len(objects))
		for i := range errs {
			errs[i] = err
		}
		return nil, errs
	}
	defer ShutdownContext()

	return fs.FSObjects.DeleteObjects(ctx, bucket, objects, opts)
}

func (fs MapRFSObjects) PutObject(ctx context.Context, bucket, object string, data *PutObjReader, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	if err := fs.checkWritePermissions(ctx, bucket, object, opts); err != nil {
		return objInfo, err
	}

	info, err := fs.FSObjects.PutObject(ctx, bucket, object, data, opts)

	if err != nil && os.IsPermission(err) {
		return info, errAccessDenied
	}

	return info, err
}

func (fs *MapRFSObjects) CopyObject(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, srcInfo ObjectInfo, srcOpts, dstOpts ObjectOptions) (oi ObjectInfo, e error) {
	if err := PrepareContext(ctx); err != nil {
		return oi, err
	}
	defer ShutdownContext()

	info, err := fs.FSObjects.CopyObject(ctx, srcBucket, srcObject, dstBucket, dstObject, srcInfo, srcOpts, dstOpts)
	if err != nil && os.IsPermission(err) {
		return info, errAccessDenied
	}

	return info, err
}

func (fs *MapRFSObjects) NewMultipartUpload(ctx context.Context, bucket, object string,
	opts ObjectOptions) (uploadID string, err error) {
	if err := fs.checkWritePermissions(ctx, bucket, object, opts); err != nil {
		return uploadID, err
	}

	return fs.FSObjects.NewMultipartUpload(ctx, bucket, object, opts)
}

func (fs *MapRFSObjects) CopyObjectPart(ctx context.Context, srcBucket, srcObject, destBucket, destObject string,
	uploadID string, partID int, startOffset int64, length int64, srcInfo ObjectInfo, srcOpts,
	dstOpts ObjectOptions) (info PartInfo, err error) {
	if err := fs.checkWritePermissions(ctx, destBucket, destObject, dstOpts); err != nil {
		return info, err
	}

	if err := fs.checkFileReadPermissions(ctx, srcBucket, srcObject, srcOpts); err != nil {
		return info, err
	}

	info, err = fs.FSObjects.CopyObjectPart(ctx, srcBucket, srcObject, destBucket, destObject, uploadID, partID, startOffset,
		length, srcInfo, srcOpts, dstOpts)
	if err != nil && os.IsPermission(err) {
		return info, errAccessDenied
	}

	return info, err
}

func (fs *MapRFSObjects) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int,
	data *PutObjReader, opts ObjectOptions) (info PartInfo, err error) {
	if err := fs.checkWritePermissions(ctx, bucket, object, opts); err != nil {
		return info, err
	}

	return fs.FSObjects.PutObjectPart(ctx, bucket, object, uploadID, partID, data, opts)
}

func (fs *MapRFSObjects) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int,
	maxParts int, opts ObjectOptions) (result ListPartsInfo, err error) {
	if err := fs.checkWritePermissions(ctx, bucket, object, opts); err != nil {
		return result, err
	}

	return fs.FSObjects.ListObjectParts(ctx, bucket, object, uploadID, partNumberMarker, maxParts, opts)
}

func (fs *MapRFSObjects) AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string, opts ObjectOptions) error {
	if err := fs.checkWritePermissions(ctx, bucket, object, ObjectOptions{}); err != nil {
		return err
	}

	return fs.FSObjects.AbortMultipartUpload(ctx, bucket, object, uploadID, opts)
}

func (fs *MapRFSObjects) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string,
	uploadedParts []CompletePart, opts ObjectOptions) (objInfo ObjectInfo, err error) {
	if err := fs.checkWritePermissions(ctx, bucket, object, opts); err != nil {
		return objInfo, err
	}

	return fs.FSObjects.CompleteMultipartUpload(ctx, bucket, object, uploadID, uploadedParts, opts)
}

func (fs MapRFSObjects) checkReadListPermissions(ctx context.Context, bucket, prefix, delimiter string) error {
	var bucketPath = path.Join(bucket, prefix)

	fullPath, err := fs.getBucketDir(ctx, bucketPath)

	if err != nil {
		return err
	}

	if delimiter != "" {
		return checkReadPermissions(fullPath, bucket, prefix)
	} else {
		return checkReadRecursivePermissions(fullPath, bucket, prefix)
	}
}

func (fs MapRFSObjects) checkWritePermissions(ctx context.Context, bucket, object string, opts ObjectOptions) error {
	uid, gid := getUidGid(ctx, opts)

	pathToObject, err := fs.getBucketDir(ctx, path.Join(bucket, object))
	if err != nil {
		return err
	}

	pathToCheck := path.Dir(pathToObject)
	for _, err := os.Stat(pathToCheck); os.IsNotExist(err); _, err = os.Stat(pathToCheck) {
		pathToCheck = path.Dir(pathToCheck)
	}

	if err := access(pathToCheck, uid, gid, W_OK); err != nil && os.IsPermission(err) {
		return PrefixAccessDenied{
			Bucket: bucket,
			Object: object,
		}
	}

	// Ignoring other errors here to make default handling work
	return nil
}

func (fs MapRFSObjects) checkFileReadPermissions(ctx context.Context, bucket, object string, opts ObjectOptions) error {
	uid, gid := getUidGid(ctx, opts)

	pathToObject, err := fs.getBucketDir(ctx, path.Join(bucket, object))
	if err != nil {
		return err
	}

	if err := access(pathToObject, uid, gid, R_OK); err != nil && os.IsPermission(err) {
		return PrefixAccessDenied{
			Bucket: bucket,
			Object: object,
		}
	}

	// Ignoring other errors here to make default handling work
	return nil
}
