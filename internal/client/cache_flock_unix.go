//go:build !windows

package client

import (
	"os"
	"syscall"
)

// flockLock 获取排他文件锁（POSIX flock）
func flockLock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// flockUnlock 释放文件锁
func flockUnlock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
