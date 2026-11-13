package tray

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Controller struct {
	assets   fs.FS
	enabled  bool
	once     sync.Once
	quitting atomic.Bool
	endLoop  func()
}

func NewController(assets embed.FS) *Controller {
	return &Controller{assets: assets, enabled: os.Getenv("SNM_DISABLE_TRAY") != "1"}
}

func (c *Controller) Enabled() bool {
	return c.enabled
}

func (c *Controller) Start(ctx context.Context, logger *slog.Logger) {
	if !c.enabled {
		logger.Info("system tray disabled by SNM_DISABLE_TRAY")
		return
	}

	c.once.Do(func() {
		start, end := systray.RunWithExternalLoop(func() {
			icon, err := fs.ReadFile(c.assets, "build/appicon.png")
			if err != nil {
				logger.Error("tray icon could not be loaded", "error", err)
			} else {
				systray.SetIcon(icon)
			}
			systray.SetTitle("Smart Network Monitoring")
			systray.SetTooltip("Smart Network Monitoring tetap berjalan di latar belakang")
			show := systray.AddMenuItem("Buka Smart Network Monitoring", "Tampilkan jendela")
			quit := systray.AddMenuItem("Keluar", "Tutup aplikasi")
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-show.ClickedCh:
						runtime.WindowShow(ctx)
						runtime.WindowUnminimise(ctx)
					case <-quit.ClickedCh:
						c.quitting.Store(true)
						runtime.Quit(ctx)
						return
					}
				}
			}()
			logger.Info("system tray started")
		}, func() {
			logger.Info("system tray stopped")
		})
		c.endLoop = end
		start()
	})
}

func (c *Controller) Stop() {
	if c.enabled && c.endLoop != nil {
		c.endLoop()
	}
}

func (c *Controller) Quitting() bool {
	return c.quitting.Load()
}
