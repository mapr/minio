package cmd

import (
	"context"
	"errors"
	"github.com/minio/minio/cmd/logger"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"
)

const W_OK uint32 = 2
const R_OK uint32 = 4

const (
	_AT_FDCWD            = -0x64
	_AT_SYMLINK_NOFOLLOW = 0x100
	_AT_EACCESS          = 0x200
)

var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

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
	RawSetfsuid(fsuid)
	if RawSetfsuid(-1) != fsuid {
		return errors.New("Failed to perform FS impersonation.")
	}

	return nil
}

func Setfsgid(fsgid int) (err error) {
	RawSetfsgid(fsgid)
	if RawSetfsgid(-1) != fsgid {
		return errors.New("Failed to perform FS impersonation")
	}

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

	if err := SetStringfsuid(uid); err != nil {
		return err
	}

	if err := SetStringfsgid(gid); err != nil {
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

func getUidGid(ctx context.Context, opts ObjectOptions) (uid, gid string) {
	info := logger.GetReqInfo(ctx)

	uid = opts.UserDefined["uid"]
	if uid == "" {
		uid = info.UID
	}

	gid = opts.UserDefined["gid"]
	if gid == "" {
		gid = info.GID
	}

	return uid, gid
}

func access(path, uidString, gidString string, mode uint32) (err error) {
	flags := _AT_EACCESS
	var st syscall.Stat_t
	if err := fstatat(_AT_FDCWD, path, &st, flags&_AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}

	mode &= 7
	if mode == 0 {
		return nil
	}

	uid, err := strconv.Atoi(uidString)
	if err != nil {
		return err
	}

	if uid == 0 {
		if mode&1 == 0 {
			// Root can read and write any file.
			return nil
		}
		if st.Mode&0111 != 0 {
			// Root can execute any file that anybody can execute.
			return nil
		}
		return syscall.EACCES
	}

	var fmode uint32
	if uint32(uid) == st.Uid {
		fmode = (st.Mode >> 6) & 7
	} else {
		gid, err := strconv.Atoi(gidString)
		if err != nil {
			return err
		}

		if uint32(gid) == st.Gid {
			fmode = (st.Mode >> 3) & 7
		} else {
			fmode = st.Mode & 7
		}
	}

	if fmode&mode == mode {
		return nil
	}

	return syscall.EACCES
}

func fstatat(fd int, path string, stat *syscall.Stat_t, flags int) (err error) {
	var _p0 *byte
	_p0, err = syscall.BytePtrFromString(path)
	if err != nil {
		return
	}
	_, _, e1 := syscall.Syscall6(syscall.SYS_NEWFSTATAT, uintptr(fd), uintptr(unsafe.Pointer(_p0)), uintptr(unsafe.Pointer(stat)), uintptr(flags), 0, 0)
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}

func checkReadRecursivePermissions(fullPath, bucket, prefix string) error {
	if err := checkReadPermissions(fullPath, bucket, prefix); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			newPath := path.Join(fullPath, file.Name())
			err := checkReadRecursivePermissions(newPath, bucket, prefix)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkReadPermissions(path, bucket, prefix string) error {
	f, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return PrefixAccessDenied{
			Bucket: bucket,
			Object: prefix,
		}
	}
	f.Close()

	// Ignoring other errors here to make default handling work
	return nil
}
