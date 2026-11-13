//go:build linux

package cryptography

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMasterKeyFallsBackWhenSecretServiceUnavailable(t *testing.T) {
	originalGet, originalSet := getKeyringSecret, setKeyringSecret
	defer func() { getKeyringSecret, setKeyringSecret = originalGet, originalSet }()
	getKeyringSecret = func(string, string) (string, error) {
		return "", errors.New("Secret Service unavailable")
	}
	setKeyringSecret = func(string, string, string) error {
		return errors.New("Secret Service unavailable")
	}
	path := filepath.Join(t.TempDir(), "master.key")
	key, err := loadMasterKey(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("master key permissions are %o, want 600", info.Mode().Perm())
	}
	if len(key) != 32 {
		t.Fatalf("master key length is %d, want 32", len(key))
	}
}

func TestLoadMasterKeyReusesFallbackWhenSecretServiceUnavailable(t *testing.T) {
	originalGet, originalSet := getKeyringSecret, setKeyringSecret
	defer func() { getKeyringSecret, setKeyringSecret = originalGet, originalSet }()
	getKeyringSecret = func(string, string) (string, error) {
		return "", errors.New("Secret Service unavailable")
	}
	setKeyringSecret = func(string, string, string) error {
		return errors.New("Secret Service unavailable")
	}
	path := filepath.Join(t.TempDir(), "master.key")
	first, err := loadMasterKey(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadMasterKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("fallback master key changed between loads")
	}
}
