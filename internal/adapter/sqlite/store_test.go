package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"snm/internal/platform/cryptography"
	"snm/internal/usecase"
	"snm/internal/usecase/port"
)

func TestOpenMigratesAndStoresTelegramTokenEncrypted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snm.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	assertDatabaseFilePermissions(t, path)

	var journalMode string
	if err := store.db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal mode is %q, want wal", journalMode)
	}
	var migrationCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 1 {
		t.Fatalf("applied %d migrations, want 1", migrationCount)
	}
	if _, err := store.db.Exec(`SELECT id, dhcp_server, is_randomized_mac FROM hosts LIMIT 0`); err != nil {
		t.Fatalf("initial schema is incomplete: %v", err)
	}

	const secret = "never-commit-this-token"
	protector, err := cryptography.NewProtector(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	settings := usecase.NewTelegramSettings(store, protector, fakeTelegramSender{})
	if err := settings.Save(context.Background(), secret, "chat-identifier", ""); err != nil {
		t.Fatal(err)
	}
	configured, chatID, _, err := settings.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !configured || chatID != "chat-identifier" {
		t.Fatalf("status returned configured=%v chat ID=%q", configured, chatID)
	}
	savedSettings, err := store.LoadTelegramSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(savedSettings.BotTokenEncrypted, []byte(secret)) {
		t.Fatal("stored Bot Token contains plaintext")
	}
	if savedSettings.ChatID != "chat-identifier" {
		t.Fatal("Chat ID was not stored")
	}
	token, err := protector.Decrypt(savedSettings.BotTokenEncrypted)
	if err != nil || string(token) != secret {
		t.Fatalf("stored credential could not be decrypted: %v", err)
	}
	dbBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	walBytes, err := os.ReadFile(path + "-wal")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if bytes.Contains(dbBytes, []byte(secret)) || bytes.Contains(walBytes, []byte(secret)) {
		t.Fatal("database contains a plaintext token")
	}
}

type fakeTelegramSender struct{}

func (fakeTelegramSender) SendTest(context.Context, string, string, string) error { return nil }

func TestCreateFirstAdminOnlyOnce(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "snm.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	if err := store.CreateFirstAdmin(ctx, "admin", "$argon2id$hash"); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateFirstAdmin(ctx, "another", "$argon2id$hash"); err != port.ErrAdminExists {
		t.Fatalf("second admin creation error = %v, want ErrAdminExists", err)
	}
}
