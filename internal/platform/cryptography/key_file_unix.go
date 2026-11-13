//go:build linux

package cryptography

import (
	"crypto/rand"
	"errors"
	"io"
	"os"
)

func readKeyFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != 32 {
		return nil, errors.New("master key file has invalid length")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, err
	}
	return data, nil
}

func createKeyFile(path string, key []byte) ([]byte, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return readKeyFile(path)
		}
		return nil, err
	}
	_, writeErr := file.Write(key)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return key, nil
}

func randomKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	return key, err
}
