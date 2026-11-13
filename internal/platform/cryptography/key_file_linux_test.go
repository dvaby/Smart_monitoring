//go:build linux

package cryptography

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFallbackMasterKeyFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "master.key")
	if _, err := createKeyFile(path, bytes.Repeat([]byte{7}, 32)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("master key permissions are %o, want 600", info.Mode().Perm())
	}
}
