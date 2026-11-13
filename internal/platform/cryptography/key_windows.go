//go:build windows

package cryptography

import (
	"crypto/rand"
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func loadMasterKey(path string) ([]byte, error) {
	// TODO-VERIFY: confirm DPAPI persistence and account scoping on Windows 10/11.
	encrypted, err := os.ReadFile(path)
	if err == nil {
		if len(encrypted) == 0 {
			return nil, errors.New("DPAPI master key file is empty")
		}
		input := windows.DataBlob{Size: uint32(len(encrypted)), Data: &encrypted[0]}
		var output windows.DataBlob
		if err := windows.CryptUnprotectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
			return nil, err
		}
		defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(output.Data))))
		key := unsafe.Slice(output.Data, output.Size)
		if len(key) != 32 {
			return nil, errors.New("DPAPI master key has invalid length")
		}
		return append([]byte(nil), key...), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	input := windows.DataBlob{Size: uint32(len(key)), Data: &key[0]}
	var output windows.DataBlob
	if err := windows.CryptProtectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(output.Data))))
	protectedKey := append([]byte(nil), unsafe.Slice(output.Data, output.Size)...)
	if err := os.WriteFile(path, protectedKey, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}
