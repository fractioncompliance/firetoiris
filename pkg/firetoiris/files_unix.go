//go:build darwin || linux

package firetoiris

import (
	"fmt"
	"os"
	"syscall"
)

func openPrivate(path string, flags int) (*os.File, error) {
	return os.OpenFile(path, flags|syscall.O_NOFOLLOW, 0600)
}
func createPrivateTemp(dir string) (*os.File, error) {
	return os.CreateTemp(dir, ".firetoiris-state-*")
}
func lockJournal(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
func unlockJournal(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
func checkPrivate(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("private_regular_file_required")
	}
	return nil
}
func replacePrivate(source, target string) error { return os.Rename(source, target) }
func syncParent(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
