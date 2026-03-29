//go:build windows

package client

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	// LOCKFILE_EXCLUSIVE_LOCK 排他锁标志
	lockfileExclusiveLock = 0x00000002
)

// flockLock 获取排他文件锁（Windows LockFileEx）
func flockLock(f *os.File) error {
	// LockFileEx(handle, flags, reserved, nNumberOfBytesToLockLow, nNumberOfBytesToLockHigh, lpOverlapped)
	var overlapped syscall.Overlapped
	r1, _, err := procLockFileEx.Call(
		uintptr(f.Fd()),
		uintptr(lockfileExclusiveLock),
		0,
		1, 0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

// flockUnlock 释放文件锁（Windows UnlockFileEx）
func flockUnlock(f *os.File) {
	var overlapped syscall.Overlapped
	procUnlockFileEx.Call(
		uintptr(f.Fd()),
		0,
		1, 0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
}
