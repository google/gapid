// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows

package file

import (
	"syscall"
	"unsafe"
)

const (
	_IO_REPARSE_TAG_MOUNT_POINT     = 0xA0000003
	_FSCTL_SET_REPARSE_POINT        = 0x900a4
	_REPARSE_MOUNTPOINT_HEADER_SIZE = 8
)

type reparseMountPointBuffer [syscall.MAXIMUM_REPARSE_DATA_BUFFER_SIZE]byte
type reparseMousePointTarget [syscall.MAXIMUM_REPARSE_DATA_BUFFER_SIZE/2 - 7]uint16

func (b *reparseMountPointBuffer) ReparseTag() *uint32 {
	return (*uint32)(unsafe.Pointer(&b[0]))
}
func (b *reparseMountPointBuffer) ReparseDataLength() *uint16 {
	return (*uint16)(unsafe.Pointer(&b[4]))
}
func (b *reparseMountPointBuffer) ReparseTargetLength() *uint16 {
	return (*uint16)(unsafe.Pointer(&b[10]))
}
func (b *reparseMountPointBuffer) ReparseTargetMaximumLength() *uint16 {
	return (*uint16)(unsafe.Pointer(&b[12]))
}
func (b *reparseMountPointBuffer) ReparseTarget() []uint16 {
	return (*reparseMousePointTarget)(unsafe.Pointer(&b[16]))[:]
}

// IsJunction returns true if the path refers to a directory junction.
func IsJunction(path Path) bool {
	pathWide, err := syscall.UTF16PtrFromString(path.System())
	if err != nil {
		return false
	}

	fd, err := syscall.CreateFile(
		pathWide,
		syscall.GENERIC_READ,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(fd)

	var rdb reparseMountPointBuffer
	var bytesReturned uint32
	err = syscall.DeviceIoControl(
		fd,
		syscall.FSCTL_GET_REPARSE_POINT,
		nil,
		0,
		&rdb[0],
		uint32(len(rdb)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return false
	}

	return *rdb.ReparseTag() == _IO_REPARSE_TAG_MOUNT_POINT
}

// Junction creates a new Junction at link pointing to target.
func Junction(link, target Path) error {
	if err := Mkdir(link); err != nil {
		return err
	}

	tgt, err := syscall.UTF16FromString(`\??\` + target.System())
	if err != nil {
		return err
	}
	lnk, err := syscall.UTF16FromString(link.System())
	if err != nil {
		return err
	}

	fd, err := syscall.CreateFile(
		&lnk[0],
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(fd)

	var rdb reparseMountPointBuffer
	reparseTargetLength := uint16(copy(rdb.ReparseTarget(), tgt)-1) * 2 // 2 == sizeof(WCHAR)
	reparseDataLength := reparseTargetLength + 12
	*rdb.ReparseTag() = _IO_REPARSE_TAG_MOUNT_POINT
	*rdb.ReparseTargetLength() = reparseTargetLength
	*rdb.ReparseTargetMaximumLength() = reparseTargetLength + 2 // 2 == sizeof(WCHAR)
	*rdb.ReparseDataLength() = reparseDataLength

	var bytesReturned uint32
	err = syscall.DeviceIoControl(
		fd,
		_FSCTL_SET_REPARSE_POINT,
		&rdb[0],
		uint32(reparseDataLength+_REPARSE_MOUNTPOINT_HEADER_SIZE),
		nil,
		0,
		&bytesReturned,
		nil,
	)

	return err
}

// Illegal characters in windows file paths.
// See https://msdn.microsoft.com/en-us/library/aa365247#NAMING_CONVENTIONS
// Excluding slashes here as we're considering paths, not filenames.
var illegalPathChars = `<>:"|?*` +
	"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f"
