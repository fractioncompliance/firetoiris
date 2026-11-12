package firetoiris

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsPrivateBundleAndBroadACLRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "private.json")
	if err := WriteBundle(path, Bundle{Version: Version}); err != nil {
		t.Fatal(err)
	}
	f, err := openPrivate(path, os.O_RDONLY)
	if err != nil {
		t.Fatal(err)
	}
	if err = checkPrivate(f); err != nil {
		t.Fatal(err)
	}
	f.Close()
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	if f, err = openPrivate(path, os.O_RDONLY); err == nil {
		f.Close()
		t.Fatal("accepted Everyone-accessible private data")
	}
}

func TestWindowsRejectsFileReparsePoint(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "private.json")
	f, err := openPrivate(target, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	link := filepath.Join(dir, "link.json")
	if err = os.Symlink(target, link); err != nil {
		t.Skip("Windows symbolic links require Developer Mode or symlink privilege; ACL and lock tests still run")
	}
	if f, err = openPrivate(link, os.O_RDONLY); err == nil {
		f.Close()
		t.Fatal("followed a reparse point")
	}
}
