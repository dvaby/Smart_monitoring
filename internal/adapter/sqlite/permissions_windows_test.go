//go:build windows

package sqlite

import (
	"os"
	"testing"
)

func assertDatabaseFilePermissions(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
