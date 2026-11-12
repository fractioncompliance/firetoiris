//go:build darwin || linux

package firetoiris

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJournalRejectsSymlinksAndPublicFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(dir, "linked")
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}
	if j, err := OpenJournal(linked); err == nil {
		j.Close()
		t.Fatal("followed state symlink")
	}
	lockPath := filepath.Join(dir, "lock-linked")
	if err := os.Symlink(target, lockPath+".lock"); err != nil {
		t.Fatal(err)
	}
	if j, err := OpenJournal(lockPath); err == nil {
		j.Close()
		t.Fatal("followed lock symlink")
	}
	if err := os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	if j, err := OpenJournal(target); err == nil {
		j.Close()
		t.Fatal("accepted public state")
	}
}
