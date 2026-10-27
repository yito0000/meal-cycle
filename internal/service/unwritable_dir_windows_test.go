//go:build windows

package service

import (
	"testing"

	"golang.org/x/sys/windows"
)

// Windows の os.Chmod は読み取り専用属性を変えるだけでフォルダへの作成を止めないため、ACL に拒否エントリを足す。
// ディレクトリでは FILE_WRITE_DATA がファイル追加、FILE_APPEND_DATA がサブフォルダ作成の権限にあたる。
func makeDirUnwritable(t *testing.T, dir string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	orig, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		t.Fatal(err)
	}
	denied, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA,
		AccessMode:        windows.DENY_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(everyone),
		},
	}}, orig)
	if err != nil {
		t.Fatal(err)
	}
	if err := setDACL(dir, denied); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = setDACL(dir, orig) })
}

func setDACL(dir string, acl *windows.ACL) error {
	return windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
