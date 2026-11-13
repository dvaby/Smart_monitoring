package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"snm/internal/usecase/port"
)

type testAdminRepository struct {
	hash       string
	failures   int
	lockedTill int64
}

func (r *testAdminRepository) AdminExists(context.Context) (bool, error) {
	return r.hash != "", nil
}

func (r *testAdminRepository) CreateFirstAdmin(_ context.Context, _, hash string) error {
	if r.hash != "" {
		return port.ErrAdminExists
	}
	r.hash = hash
	return nil
}

func (r *testAdminRepository) AdminLoginState(context.Context, string) (string, int, int64, error) {
	if r.hash == "" {
		return "", 0, 0, port.ErrAdminNotFound
	}
	return r.hash, r.failures, r.lockedTill, nil
}

func (r *testAdminRepository) RecordAdminFailure(_ context.Context, _ string, failures int, lockedTill int64) error {
	r.failures, r.lockedTill = failures, lockedTill
	return nil
}

func (r *testAdminRepository) ClearAdminFailures(context.Context, string) error {
	r.failures, r.lockedTill = 0, 0
	return nil
}

type testHasher struct{}

func (testHasher) Hash(string) (string, error) { return "valid-hash", nil }
func (testHasher) Verify(hash, password string) bool {
	return hash == "valid-hash" && password == "correct-password"
}

func TestAuthenticationLocksAdminAfterFiveFailures(t *testing.T) {
	repository := &testAdminRepository{hash: "valid-hash"}
	auth := NewAuthentication(repository, testHasher{})
	ctx := context.Background()

	for attempt := 0; attempt < 5; attempt++ {
		if err := auth.Authenticate(ctx, "admin", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d error = %v, want invalid credentials", attempt+1, err)
		}
	}
	if repository.lockedTill <= time.Now().UTC().Unix() {
		t.Fatal("account was not locked after five failed attempts")
	}
	if err := auth.Authenticate(ctx, "admin", "correct-password"); err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("locked account should report temporary lock, got %v", err)
	}
}

func TestCreateFirstAdminRequiresLongPassword(t *testing.T) {
	auth := NewAuthentication(&testAdminRepository{}, testHasher{})
	if err := auth.CreateFirstAdmin(context.Background(), "admin", "short"); err == nil {
		t.Fatal("short admin password should be rejected")
	}
}
