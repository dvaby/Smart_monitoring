package usecase

import (
	"context"
	"errors"
	"strings"

	"snm/internal/usecase/port"
)

var ErrTelegramSettingsIncomplete = errors.New("bot token and chat ID are required")

type TelegramSettings struct {
	repository port.NotificationSettingsRepository
	protector  port.SecretProtector
	sender     port.TelegramSender
}

func NewTelegramSettings(repository port.NotificationSettingsRepository, protector port.SecretProtector, sender port.TelegramSender) *TelegramSettings {
	return &TelegramSettings{repository: repository, protector: protector, sender: sender}
}

func (s *TelegramSettings) Save(ctx context.Context, token, chatID, proxyURL string) error {
	token = strings.TrimSpace(token)
	chatID = strings.TrimSpace(chatID)
	current, err := s.repository.LoadTelegramSettings(ctx)
	if err != nil {
		return err
	}
	if token == "" {
		if len(current.BotTokenEncrypted) == 0 {
			return ErrTelegramSettingsIncomplete
		}
	} else {
		current.BotTokenEncrypted, err = s.protector.Encrypt([]byte(token))
		if err != nil {
			return err
		}
	}
	if chatID == "" {
		chatID = current.ChatID
	}
	if chatID == "" {
		return ErrTelegramSettingsIncomplete
	}
	if strings.TrimSpace(proxyURL) != "" {
		current.ProxyEncrypted, err = s.protector.Encrypt([]byte(strings.TrimSpace(proxyURL)))
		if err != nil {
			return err
		}
	} else {
		current.ProxyEncrypted = nil
	}
	return s.repository.SaveTelegramSettings(ctx, current.BotTokenEncrypted, chatID, current.ProxyEncrypted)
}

func (s *TelegramSettings) Status(ctx context.Context) (bool, string, bool, error) {
	settings, err := s.repository.LoadTelegramSettings(ctx)
	if err != nil {
		return false, "", false, err
	}
	return len(settings.BotTokenEncrypted) > 0, settings.ChatID, len(settings.ProxyEncrypted) > 0, nil
}

func (s *TelegramSettings) Test(ctx context.Context, tokenOverride, chatIDOverride, proxyOverride string) error {
	settings, err := s.repository.LoadTelegramSettings(ctx)
	if err != nil {
		return err
	}
	tokenValue := tokenOverride
	if tokenValue == "" {
		if len(settings.BotTokenEncrypted) == 0 {
			return ErrTelegramSettingsIncomplete
		}
		token, err := s.protector.Decrypt(settings.BotTokenEncrypted)
		if err != nil {
			return err
		}
		tokenValue = string(token)
	}
	chatID := settings.ChatID
	if chatIDOverride != "" {
		chatID = chatIDOverride
	}
	proxyURL := proxyOverride
	if len(settings.ProxyEncrypted) > 0 {
		if proxyURL == "" {
			proxy, decryptErr := s.protector.Decrypt(settings.ProxyEncrypted)
			if decryptErr != nil {
				return decryptErr
			}
			proxyURL = string(proxy)
		}
	}
	if tokenValue == "" || chatID == "" {
		return ErrTelegramSettingsIncomplete
	}
	return s.sender.SendTest(ctx, tokenValue, chatID, proxyURL)
}
