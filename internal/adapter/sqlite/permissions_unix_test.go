//go:build !windows

package sqlite

import (
	"os"
	"testing"
)

func assertDatabaseFilePermissions(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("database permissions are %o, want 600", info.Mode().Perm())
	}
}
