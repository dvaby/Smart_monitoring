package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Protector struct {
	aead cipher.AEAD
}

func NewProtector(dataDir string) (*Protector, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	key, err := loadMasterKey(filepath.Join(dataDir, "master.key"))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Protector{aead: aead}, nil
}

func (p *Protector) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return p.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (p *Protector) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := p.aead.NonceSize()
	if len(ciphertext) < nonceSize+p.aead.Overhead() {
		return nil, errors.New("encrypted value is truncated")
	}
	nonce, body := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := p.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}
	return plaintext, nil
}
