package firetoiris

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

func privateDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	sid := user.User.Sid.String()
	return windows.SecurityDescriptorFromString("O:" + sid + "D:P(A;;FA;;;SY)(A;;FA;;;" + sid + ")")
}

func openPrivate(path string, flags int) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	sd, err := privateDescriptor()
	if err != nil {
		return nil, err
	}
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	access := uint32(windows.GENERIC_READ | windows.READ_CONTROL)
	if flags&(os.O_WRONLY|os.O_RDWR) != 0 {
		access |= windows.GENERIC_WRITE
	}
	creation := uint32(windows.OPEN_EXISTING)
	if flags&os.O_CREATE != 0 {
		creation = windows.OPEN_ALWAYS
		if flags&os.O_EXCL != 0 {
			creation = windows.CREATE_NEW
		}
	}
	handle, err := windows.CreateFile(name, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, &attributes, creation, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	f := os.NewFile(uintptr(handle), path)
	var info windows.ByHandleFileInformation
	err = windows.GetFileInformationByHandle(handle, &info)
	if err == nil && info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		err = fmt.Errorf("regular_file_without_reparse_point_required")
	}
	if err == nil {
		err = checkPrivate(f)
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func checkPrivate(f *os.File) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	descriptor, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil || owner == nil || !owner.Equals(user.User.Sid) {
		return fmt.Errorf("current_user_owner_required")
	}
	acl, _, err := descriptor.DACL()
	if err != nil || acl == nil {
		return fmt.Errorf("private_DACL_required")
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err = windows.GetAce(acl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return fmt.Errorf("unsupported_DACL_entry")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.Equals(user.User.Sid) && sid.String() != "S-1-5-18" {
			return fmt.Errorf("file_access_must_be_current_user_and_SYSTEM_only")
		}
	}
	return nil
}
func createPrivateTemp(dir string) (*os.File, error) {
	for i := 0; i < 10; i++ {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, err
		}
		f, err := openPrivate(filepath.Join(dir, ".firetoiris-state-"+hex.EncodeToString(random[:])), os.O_CREATE|os.O_EXCL|os.O_RDWR)
		if os.IsExist(err) {
			continue
		}
		return f, err
	}
	return nil, fmt.Errorf("private_temp_creation_failed")
}
func lockJournal(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
}
func unlockJournal(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{})
}
func replacePrivate(source, target string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func syncParent(string) error { return nil }
