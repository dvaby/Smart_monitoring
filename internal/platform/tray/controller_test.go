package tray

import (
	"context"
	"embed"
	"io"
	"log/slog"
	"testing"
)

func TestNewControllerCanDisableTray(t *testing.T) {
	t.Setenv("SNM_DISABLE_TRAY", "1")
	controller := NewController(embed.FS{})
	if controller.Enabled() {
		t.Fatal("tray should be disabled")
	}
	controller.Start(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	controller.Stop()
}

func TestNewControllerEnablesTrayByDefault(t *testing.T) {
	t.Setenv("SNM_DISABLE_TRAY", "")
	controller := NewController(embed.FS{})
	if !controller.Enabled() {
		t.Fatal("tray should be enabled by default")
	}
}
