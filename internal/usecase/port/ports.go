package port

import (
	"context"
	"errors"
	"time"
)

var ErrAdminExists = errors.New("administrator account already exists")
var ErrAdminNotFound = errors.New("administrator account not found")

type AdminRepository interface {
	AdminExists(context.Context) (bool, error)
	CreateFirstAdmin(context.Context, string, string) error
	AdminLoginState(context.Context, string) (string, int, int64, error)
	RecordAdminFailure(context.Context, string, int, int64) error
	ClearAdminFailures(context.Context, string) error
}

type NotificationSettingsRepository interface {
	SaveTelegramSettings(context.Context, []byte, string, []byte) error
	LoadTelegramSettings(context.Context) (TelegramSettings, error)
}

type TelegramSettings struct {
	BotTokenEncrypted []byte
	ChatID            string
	ProxyEncrypted    []byte
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) bool
}

type SecretProtector interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type TelegramSender interface {
	SendTest(context.Context, string, string, string) error
}

type Clock interface {
	Now() time.Time
}
