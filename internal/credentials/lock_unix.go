//go:build !windows

package credentials

import (
	"os"
	"syscall"
)

func lockFile(f *os.File) error {
	// #nosec G115 -- file descriptors fit in an int on all supported platforms
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func unlockFile(f *os.File) error {
	// #nosec G115 -- file descriptors fit in an int on all supported platforms
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
