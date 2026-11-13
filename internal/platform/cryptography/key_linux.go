//go:build linux

package cryptography

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "SmartNetworkMonitoring"
	keyringAccount = "master-key"
)

var (
	getKeyringSecret = keyring.Get
	setKeyringSecret = keyring.Set
)

func loadMasterKey(path string) ([]byte, error) {
	// TODO-VERIFY: confirm Secret Service unlock and persistence on the target Ubuntu desktop.
	encoded, err := getKeyringSecret(keyringService, keyringAccount)
	if err == nil {
		if key, decodeErr := base64.RawStdEncoding.DecodeString(encoded); decodeErr == nil && len(key) == 32 {
			return key, nil
		}
		return nil, errors.New("Secret Service master key has invalid format")
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		key, keyFileErr := readKeyFile(path)
		if keyFileErr == nil {
			return key, nil
		}
		if !errors.Is(keyFileErr, os.ErrNotExist) {
			return nil, keyFileErr
		}
		key = make([]byte, 32)
		if _, randomErr := rand.Read(key); randomErr != nil {
			return nil, randomErr
		}
		return createKeyFile(path, key)
	}
	key, err := readKeyFile(path)
	if err == nil {
		if err := setKeyringSecret(keyringService, keyringAccount, base64.RawStdEncoding.EncodeToString(key)); err != nil {
			return nil, fmt.Errorf("store existing master key in Secret Service: %w", err)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := setKeyringSecret(keyringService, keyringAccount, base64.RawStdEncoding.EncodeToString(key)); err == nil {
		return key, nil
	}
	return createKeyFile(path, key)
}
