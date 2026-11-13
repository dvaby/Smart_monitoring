package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"snm/internal/usecase/port"
	"snm/migrations"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, err
	}
	dsn := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	dsn += "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		return err
	}
	files, err := migrations.Files.ReadDir(".")
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, file := range files {
		var version int
		if _, err := fmt.Sscanf(file.Name(), "%d_", &version); err != nil {
			return fmt.Errorf("invalid migration filename %q: %w", file.Name(), err)
		}
		var applied bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		script, err := migrations.Files.ReadFile(file.Name())
		if err != nil {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(script)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`, version, time.Now().UTC().Unix())
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", file.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO notification_settings(id) VALUES (1)`)
	return err
}

func (s *Store) AdminExists(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')`).Scan(&exists)
	return exists, err
}

func (s *Store) CreateFirstAdmin(ctx context.Context, username, passwordHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return port.ErrAdminExists
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO users(username, password_hash, role, created_at) VALUES (?, ?, 'admin', ?)`,
		username, passwordHash, time.Now().UTC().Unix())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AdminLoginState(ctx context.Context, username string) (string, int, int64, error) {
	var hash string
	var failedAttempts int
	var lockedUntil sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT password_hash, failed_attempts, locked_until FROM users WHERE username = ? AND role = 'admin' AND is_active = 1`, username).Scan(&hash, &failedAttempts, &lockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, 0, port.ErrAdminNotFound
	}
	if err != nil {
		return "", 0, 0, err
	}
	if lockedUntil.Valid {
		return hash, failedAttempts, lockedUntil.Int64, nil
	}
	return hash, failedAttempts, 0, nil
}

func (s *Store) RecordAdminFailure(ctx context.Context, username string, failures int, lockedUntil int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET failed_attempts = ?, locked_until = ? WHERE username = ? AND role = 'admin'`,
		failures, nullableInt(lockedUntil), username)
	return err
}

func (s *Store) ClearAdminFailures(ctx context.Context, username string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE username = ? AND role = 'admin'`, username)
	return err
}

func nullableInt(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func (s *Store) SaveTelegramSettings(ctx context.Context, tokenEncrypted []byte, chatID string, proxyEncrypted []byte) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notification_settings
		SET telegram_enabled = 1, telegram_bot_token_enc = ?, telegram_chat_id = ?, telegram_proxy_enc = ?
		WHERE id = 1`, tokenEncrypted, chatID, proxyEncrypted)
	return err
}

func (s *Store) LoadTelegramSettings(ctx context.Context) (port.TelegramSettings, error) {
	var settings port.TelegramSettings
	var token, proxy []byte
	var chatID sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT telegram_bot_token_enc, telegram_chat_id, telegram_proxy_enc
		FROM notification_settings WHERE id = 1`).Scan(&token, &chatID, &proxy)
	if err != nil {
		return settings, err
	}
	settings.BotTokenEncrypted = token
	settings.ProxyEncrypted = proxy
	if chatID.Valid {
		settings.ChatID = strings.TrimSpace(chatID.String)
	}
	return settings, nil
}
