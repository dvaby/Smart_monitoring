package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"snm/internal/usecase/port"
)

var ErrInvalidCredentials = errors.New("username or password is incorrect")

type Authentication struct {
	admins port.AdminRepository
	hasher port.PasswordHasher
}

func NewAuthentication(admins port.AdminRepository, hasher port.PasswordHasher) *Authentication {
	return &Authentication{admins: admins, hasher: hasher}
}

func (a *Authentication) AdminExists(ctx context.Context) (bool, error) {
	return a.admins.AdminExists(ctx)
}

func (a *Authentication) CreateFirstAdmin(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username is required")
	}
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	hash, err := a.hasher.Hash(password)
	if err != nil {
		return err
	}
	err = a.admins.CreateFirstAdmin(ctx, username, hash)
	if errors.Is(err, port.ErrAdminExists) {
		return port.ErrAdminExists
	}
	return err
}

func (a *Authentication) Authenticate(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	hash, failures, lockedUntil, err := a.admins.AdminLoginState(ctx, username)
	if errors.Is(err, port.ErrAdminNotFound) {
		return ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	if lockedUntil > now {
		return errors.New("akun terkunci sementara karena terlalu banyak percobaan login")
	}
	if !a.hasher.Verify(hash, password) {
		failures++
		lockUntil := int64(0)
		if failures >= 5 {
			lockUntil = now + int64((15 * time.Minute).Seconds())
		}
		if err := a.admins.RecordAdminFailure(ctx, username, failures, lockUntil); err != nil {
			return err
		}
		return ErrInvalidCredentials
	}
	if err := a.admins.ClearAdminFailures(ctx, username); err != nil {
		return err
	}
	return nil
}
