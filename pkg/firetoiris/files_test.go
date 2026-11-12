package firetoiris

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateFilesAndReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "private.json")
	f, err := openPrivate(path, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	if err = checkPrivate(f); err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString("original"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	staged, err := createPrivateTemp(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(staged.Name())
	if err = checkPrivate(staged); err != nil {
		t.Fatal(err)
	}
	if _, err = staged.WriteString("replacement"); err != nil {
		t.Fatal(err)
	}
	if err = staged.Sync(); err != nil {
		t.Fatal(err)
	}
	staged.Close()
	if err = replacePrivate(staged.Name(), path); err != nil {
		t.Fatal(err)
	}
	f, err = openPrivate(path, os.O_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = checkPrivate(f); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "replacement" {
		t.Fatal("replacement lost data", err)
	}
}
