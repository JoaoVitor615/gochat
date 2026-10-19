//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")
)

func detectOSLocale() string {
	if err := procGetUserDefaultLocaleName.Find(); err != nil {
		return ""
	}
	const maxLocaleNameLen = 85
	buffer := make([]uint16, maxLocaleNameLen)
	ret, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(maxLocaleNameLen),
	)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buffer)
}
