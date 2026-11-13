package userdirs

import (
	"os"
	"path/filepath"
)

func DataDirectory() (string, error) {
	if configured := os.Getenv("SNM_DATA_DIR"); configured != "" {
		return filepath.Abs(configured)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "SmartNetworkMonitoring"), nil
}
