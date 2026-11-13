package wailsbind

import (
	"context"
	"errors"
	"os"
	"sync"

	"snm/internal/usecase"
	"snm/internal/usecase/port"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	auth     *usecase.Authentication
	telegram *usecase.TelegramSettings
	ctx      context.Context
	mu       sync.RWMutex
	loggedIn bool
}

type Bindings struct {
	app *App
}

type Lifecycle struct {
	app *App
}

type State struct {
	AdminConfigured    bool   `json:"adminConfigured"`
	Authenticated      bool   `json:"authenticated"`
	TelegramConfigured bool   `json:"telegramConfigured"`
	ProxyConfigured    bool   `json:"proxyConfigured"`
	ChatID             string `json:"chatId"`
}

func New(auth *usecase.Authentication, telegram *usecase.TelegramSettings) (*Bindings, *Lifecycle) {
	app := &App{auth: auth, telegram: telegram}
	return &Bindings{app: app}, &Lifecycle{app: app}
}

func (l *Lifecycle) Startup(ctx context.Context) {
	a := l.app
	a.ctx = ctx
}

func (l *Lifecycle) Shutdown() {
	a := l.app
	a.mu.Lock()
	a.loggedIn = false
	a.mu.Unlock()
}

func (b *Bindings) GetState() (State, error) {
	a := b.app
	adminConfigured, err := a.auth.AdminExists(context.Background())
	if err != nil {
		return State{}, err
	}
	state := State{AdminConfigured: adminConfigured}
	a.mu.RLock()
	state.Authenticated = a.loggedIn
	a.mu.RUnlock()
	if state.Authenticated {
		state.TelegramConfigured, state.ChatID, state.ProxyConfigured, err = a.telegram.Status(context.Background())
		if err != nil {
			return State{}, err
		}
	}
	return state, nil
}

func (b *Bindings) CreateAdmin(username, password string) error {
	a := b.app
	exists, err := a.auth.AdminExists(context.Background())
	if err != nil {
		return err
	}
	if exists {
		return port.ErrAdminExists
	}
	if err := a.auth.CreateFirstAdmin(context.Background(), username, password); err != nil {
		return err
	}
	a.setAuthenticated(true)
	return nil
}

func (b *Bindings) Login(username, password string) error {
	a := b.app
	if err := a.auth.Authenticate(context.Background(), username, password); err != nil {
		return err
	}
	a.setAuthenticated(true)
	return nil
}

func (b *Bindings) Logout() {
	a := b.app
	a.setAuthenticated(false)
}

func (b *Bindings) SaveTelegramSettings(token, chatID, proxyURL string) error {
	a := b.app
	if !a.authenticated() {
		return errors.New("silakan login sebagai admin terlebih dahulu")
	}
	return a.telegram.Save(context.Background(), token, chatID, proxyURL)
}

func (b *Bindings) TestTelegram(token, chatID, proxyURL string) (string, error) {
	a := b.app
	if !a.authenticated() {
		return "", errors.New("silakan login sebagai admin terlebih dahulu")
	}
	if err := a.telegram.Test(context.Background(),
		firstNonEmpty(token, os.Getenv("SNM_TELEGRAM_BOT_TOKEN")),
		firstNonEmpty(chatID, os.Getenv("SNM_TELEGRAM_CHAT_ID")),
		firstNonEmpty(proxyURL, os.Getenv("SNM_TELEGRAM_PROXY_URL"))); err != nil {
		return "", err
	}
	return "Pesan uji berhasil dikirim ke Telegram.", nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (b *Bindings) HideWindow() {
	a := b.app
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

func (b *Bindings) ShowWindow() {
	a := b.app
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
	}
}

func (a *App) authenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.loggedIn
}

func (a *App) setAuthenticated(value bool) {
	a.mu.Lock()
	a.loggedIn = value
	a.mu.Unlock()
}
