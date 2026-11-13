package cryptography

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func TestProtectorRoundTripAndAuthentication(t *testing.T) {
	block, err := aes.NewCipher(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	protector := &Protector{aead: aead}
	secret := []byte("never-store-this-in-plaintext")
	encrypted, err := protector.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, secret) {
		t.Fatal("ciphertext contains plaintext")
	}
	decoded, err := protector.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, secret) {
		t.Fatal("decrypted value differs from original")
	}
	encrypted[len(encrypted)-1] ^= 0xff
	if _, err := protector.Decrypt(encrypted); err == nil {
		t.Fatal("tampered ciphertext should fail authentication")
	}
}
