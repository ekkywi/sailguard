//go:build windows

package windows

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func lookupProcessUser(pid uint32) string {
	if pid == 0 {
		return ""
	}

	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		h, err = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil {
			return ""
		}
	}
	defer windows.CloseHandle(h)

	var token windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token); err != nil {
		return ""
	}
	defer token.Close()

	tu, err := token.GetTokenUser()
	if err != nil || tu == nil || tu.User.Sid == nil {
		return ""
	}

	account, domain, _, err := tu.User.Sid.LookupAccount("")
	if err != nil {
		return ""
	}
	if domain == "" {
		return account
	}
	return fmt.Sprintf(`%s\%s`, domain, account)
}