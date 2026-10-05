//go:build windows
// +build windows

package downloader

import (
	"syscall"
	"time"
)

func setCreationAndModificationTime(p string, t time.Time) error {
	path16, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return err
	}
	h, err := syscall.CreateFile(path16,
		syscall.FILE_WRITE_ATTRIBUTES,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)

	ft := syscall.NsecToFiletime(t.UnixNano())
	// SetFileTime takes CreationTime, LastAccessTime, LastWriteTime
	return syscall.SetFileTime(h, &ft, &ft, &ft)
}
