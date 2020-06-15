package cmd

import (
	"context"
	"errors"
	"github.com/minio/minio/cmd/logger"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
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

	return fs.FSObjects.ListObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
}

func (fs MapRFSObjects) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	if err := PrepareContext(ctx); err != nil {
		return ListObjectsV2Info{}, err
	}
	defer ShutdownContext()

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
	info, err := fs.FSObjects.PutObject(ctx, bucket, object, data, opts)

	if err != nil && strings.Contains(err.Error(), "permission denied") {
		return info, errAccessDenied
	}

	return info, err
}

func (fs MapRFSObjects) IsCompressionSupported() bool {
	return fs.FSObjects.IsCompressionSupported()
}

func (fs MapRFSObjects) IsEncryptionSupported() bool {
	return fs.FSObjects.IsEncryptionSupported()
}

func RawSetfsuid(fsuid int) (prevFsuid int) {
	r1, _, _ := syscall.RawSyscall(syscall.SYS_SETFSUID, uintptr(fsuid), 0, 0)
	return int(r1)
}

func RawSetfsgid(fsgid int) (prevFsgid int) {
	r1, _, _ := syscall.RawSyscall(syscall.SYS_SETFSGID, uintptr(fsgid), 0, 0)
	return int(r1)
}

func SetStringfsuid(fsuid string) (err error) {
	if fsuid == "" {
		return nil
	}

	uid, err := strconv.Atoi(fsuid)
	if err != nil {
		return err
	}

	return Setfsuid(uid)
}

func SetStringfsgid(fsgid string) (err error) {
	if fsgid == "" {
		return nil
	}

	gid, err := strconv.Atoi(fsgid)
	if err != nil {
		return err
	}

	return Setfsgid(gid)
}

func Setfsuid(fsuid int) (err error) {
	logger.Info("Setting UID to " + strconv.Itoa(fsuid))
	RawSetfsuid(fsuid)
	if RawSetfsuid(-1) != fsuid {
		return errors.New("Failed to perform FS impersonation.")
	}

	logger.Info("UID is " + strconv.Itoa(fsuid))

	return nil
}

func Setfsgid(fsgid int) (err error) {
	logger.Info("Setting GID to " + strconv.Itoa(fsgid))
	RawSetfsgid(fsgid)
	if RawSetfsgid(-1) != fsgid {
		return errors.New("Failed to perform FS impersonation")
	}

	logger.Info("GID is " + strconv.Itoa(fsgid))

	return nil
}

func PrepareContext(ctx context.Context) error {
	reqInfo := logger.GetReqInfo(ctx)

	err := PrepareContextUidGid(reqInfo.UID, reqInfo.GID)
	if err != nil {
		logger.LogIf(ctx, err)
	}

	return err
}

func PrepareContextUidGid(uid, gid string) error {
	runtime.LockOSThread()

	if err := SetStringfsuid(gid); err != nil {
		return err
	}

	if err := SetStringfsgid(uid); err != nil {
		return err
	}

	return nil
}

func ShutdownContext() error {
	defer runtime.UnlockOSThread()

	if err := Setfsuid(syscall.Geteuid()); err != nil {
		return err
	}

	if err := Setfsgid(syscall.Getegid()); err != nil {
		return err
	}

	return nil
}
