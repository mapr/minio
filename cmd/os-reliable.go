/*
 * MinIO Cloud Storage, (C) 2018 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
)

// Wrapper functions to os.RemoveAll, which calls reliableRemoveAll
// this is to ensure that if there is a racy parent directory
// create in between we can simply retry the operation.
func removeAll(dirPath string) (err error) {
	if dirPath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(dirPath); err != nil {
		return err
	}

	if err = reliableRemoveAll(dirPath); err != nil {
		switch {
		case isSysErrNotDir(err):
			// File path cannot be verified since one of
			// the parents is a file.
			return errFileAccessDenied
		case isSysErrPathNotFound(err):
			// This is a special case should be handled only for
			// windows, because windows API does not return "not a
			// directory" error message. Handle this specifically
			// here.
			return errFileAccessDenied
		}
	}
	return err
}

// Reliably retries os.RemoveAll if for some reason os.RemoveAll returns
// syscall.ENOTEMPTY (children has files).
func reliableRemoveAll(dirPath string) (err error) {
	i := 0
	for {
		// Removes all the directories and files.
		if err = os.RemoveAll(dirPath); err != nil {
			// Retry only for the first retryable error.
			if isSysErrNotEmpty(err) && i == 0 {
				i++
				continue
			}
		}
		break
	}
	return err
}

// Wrapper functions to os.MkdirAll, which calls reliableMkdirAll
// this is to ensure that if there is a racy parent directory
// delete in between we can simply retry the operation.
func mkdirAll(dirPath string, mode os.FileMode) (err error) {
	if dirPath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(dirPath); err != nil {
		return err
	}

	if err = reliableMkdirAll(dirPath, mode); err != nil {
		// File path cannot be verified since one of the parents is a file.
		if isSysErrNotDir(err) {
			return errFileAccessDenied
		} else if isSysErrPathNotFound(err) {
			// This is a special case should be handled only for
			// windows, because windows API does not return "not a
			// directory" error message. Handle this specifically here.
			return errFileAccessDenied
		}
	}
	return err
}

// Reliably retries os.MkdirAll if for some reason os.MkdirAll returns
// syscall.ENOENT (parent does not exist).
func reliableMkdirAll(dirPath string, mode os.FileMode) (err error) {
	i := 0
	for {
		// Creates all the parent directories, with mode 0777 mkdir honors system umask.
		if err = os.MkdirAll(dirPath, mode); err != nil {
			// Retry only for the first retryable error.
			if osIsNotExist(err) && i == 0 {
				i++
				continue
			}
		}
		break
	}
	return err
}

// Reliably retries os.MkdirAll if for some reason os.MkdirAll returns
// syscall.ENOENT (parent does not exist).
func reliableMkdirAllWithUidGid(dirPath, uid, gid string, mode os.FileMode) (err error) {
	if _, err = os.Stat(dirPath); !os.IsNotExist(err) {
		return nil
	}

	if uid == "" || gid == "" {
		return reliableMkdirAll(dirPath, mode)
	}

	parts := strings.Split(dirPath, "/")
	var newPath = ""
	for _, part := range parts {
		newPath += part + "/"
		if _, err = os.Stat(newPath); os.IsNotExist(err) {
			for i := 0; i < 2; i++ {
				// Creates all the parent directories, with mode mkdir honors system umask.
				if err = os.Mkdir(newPath, mode); err != nil {
					// Retry only for the first retryable error.
					if os.IsNotExist(err) {
						continue
					}
				} else if err = chown(newPath, uid, gid); err != nil {
					continue
				}
				break
			}
		} else {
			continue
		}

		if err != nil {
			break
		}
	}

	return err
}

// Chown specified directory
func chown(path, uid, gid string) error {
	if uid == "" || gid == "" {
		return nil
	}

	numberUid, err := strconv.Atoi(uid)
	if err != nil {
		return err
	}

	numberGid, err := strconv.Atoi(gid)
	if err != nil {
		return err
	}

	if err := syscall.Chown(path, numberUid, numberGid); err != nil {
		return err
	}

	return nil
}

// Wrapper function to os.Rename, which calls reliableMkdirAll
// and reliableRenameAll. This is to ensure that if there is a
// racy parent directory delete in between we can simply retry
// the operation.
func renameAll(srcFilePath, dstFilePath string) (err error) {
	return renameAllWithUidGid(srcFilePath, dstFilePath, "", "")
}

// Wrapper function to os.Rename, which calls reliableMkdirAll
// and reliableRenameAll. This is to ensure that if there is a
// racy parent directory delete in between we can simply retry
// the operation with specified UID and GID.
func renameAllWithUidGid(srcFilePath, dstFilePath, uid, gid string) (err error) {
	if srcFilePath == "" || dstFilePath == "" {
		return errInvalidArgument
	}

	if err = checkPathLength(srcFilePath); err != nil {
		return err
	}
	if err = checkPathLength(dstFilePath); err != nil {
		return err
	}

	if err = reliableRenameWithUidGid(srcFilePath, dstFilePath, uid, gid); err != nil {
		switch {
		case isSysErrNotDir(err) && !osIsNotExist(err):
			// Windows can have both isSysErrNotDir(err) and osIsNotExist(err) returning
			// true if the source file path contains an inexistant directory. In that case,
			// we want to return errFileNotFound instead, which will honored in subsequent
			// switch cases
			return errFileAccessDenied
		case isSysErrPathNotFound(err):
			// This is a special case should be handled only for
			// windows, because windows API does not return "not a
			// directory" error message. Handle this specifically here.
			return errFileAccessDenied
		case isSysErrCrossDevice(err):
			return fmt.Errorf("%w (%s)->(%s)", errCrossDeviceLink, srcFilePath, dstFilePath)
		case osIsNotExist(err):
			return errFileNotFound
		case osIsExist(err):
			// This is returned only when destination is a directory and we
			// are attempting a rename from file to directory.
			return errIsNotRegular
		default:
			return err
		}
	}
	return nil
}

// Reliably retries os.RenameAll if for some reason os.RenameAll returns
// syscall.ENOENT (parent does not exist).
func reliableRename(srcFilePath, dstFilePath string) (err error) {
	return reliableRenameWithUidGid(srcFilePath, dstFilePath, "", "")
}

// Reliably retries os.RenameAll if for some reason os.RenameAll returns
// syscall.ENOENT (parent does not exist) with specified UID and GID.
func reliableRenameWithUidGid(srcFilePath, dstFilePath, uid, gid string) (err error) {
	if err = reliableMkdirAllWithUidGid(path.Dir(dstFilePath), uid, gid, 0755); err != nil {
		return err
	}

	for i := 0; i < 2; i++ {
		// After a successful parent directory create attempt a renameAll.
		if err = os.Rename(srcFilePath, dstFilePath); err != nil {
			// Retry only for the first retryable error.
			if osIsNotExist(err) {
				continue
			}
		} else if err = chown(dstFilePath, uid, gid); err != nil {
			continue
		}
		break
	}
	return err
}
