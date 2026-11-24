package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/app/iconpng"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/agentcli"
	"github.com/acam-infra/parla-viva/internal/applog"
	"github.com/acam-infra/parla-viva/internal/appui"
	"github.com/acam-infra/parla-viva/internal/assets"
	"github.com/acam-infra/parla-viva/internal/config"
	"github.com/acam-infra/parla-viva/internal/dc"
)

func init() {
	// Pin main goroutine to OS thread 0 before any I/O so macOS AppKit/Metal
	// always runs on the Cocoa main thread and never deadlocks on dispatch_sync.
	runtime.LockOSThread()
}

func main() {
	if agentcli.ShouldHandle(os.Args[1:]) {
		os.Exit(agentcli.Run(context.Background(), os.Args[1:], agentcli.Options{}))
	}

	demoFlag := flag.Bool("demo", false, "run in offline preview mode without deltachat-rpc-server")
	rpcPath := flag.String("rpc", "", "explicit path to deltachat-rpc-server binary")
	accountsPath := flag.String("accounts", dc.DefaultAccountsDir(), "Delta Chat accounts directory")
	flag.Parse()

	inv := new(app.Invalidator)
	webHost := &app.WebHost{}
	var model *appui.Model
	var transport *dc.Transport
	var ipcSrv *dc.IPCServer

	appCfg, cfgPath, _ := config.Load(*accountsPath)
	logger, _ := applog.Open(appCfg.LogFile, appCfg.LogLevel, appCfg.LogMaxMB)
	if logger != nil {
		defer logger.Close()
		logger.Info("startup", "version", agentcli.Version, "accountsDir", *accountsPath, "config", cfgPath, "logLevel", appCfg.LogLevel)
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	effRPC := *rpcPath
	if effRPC == "" {
		effRPC = appCfg.RPCServer
	}

	if !*demoFlag {
		if bin := dc.DiscoverRPCServer(effRPC); bin != "" {
			transport = dc.NewTransport()
			if err := transport.StartProcess(bin, *accountsPath); err == nil {
				client := dc.NewClient(transport)
				model = appui.NewLiveModel(client, inv)
				model.AccountsPath = *accountsPath
				model.EnterToSend = appCfg.EnterToSend
				if appCfg.Theme == "dark" {
					model.SetDarkMode(true)
				}
				switch appCfg.Layout {
				case "workspace":
					model.Layout = appui.LayoutWorkspace
				case "compact":
					model.Layout = appui.LayoutCompact
				}
				if !appCfg.Notifications && model.Notifier != nil {
					model.Notifier.Close()
					model.Notifier = nil
				}
				cacheHit := model.LoadCache(*accountsPath)
				if srv, ierr := dc.StartIPCServer(transport, *accountsPath); ierr == nil {
					ipcSrv = srv
					model.OnEvent = srv.PublishEvent
					if logger != nil {
						logger.Info("ipc_ready", "addr", srv.Addr(), "cacheHit", cacheHit, "rpcServer", bin)
					}
				}
				model.StartLiveSync(ctx)
			} else {
				if logger != nil {
					logger.Error("rpc_start_failed", "bin", bin, "err", err)
				}
				log.Printf("deltachat-rpc-server start failed (%v); falling back to demo mode", err)
			}
		}
	}

	if model == nil {
		model = appui.NewDemoModel()
		model.Invalidator = inv
	}
	model.WebHost = webHost
	webHost.OnReady = func() {
		model.WebReady = true
	}

	var statusMu sync.Mutex
	var statusItem *app.StatusItem
	removeStatusItem := func() {
		statusMu.Lock()
		item := statusItem
		statusItem = nil
		statusMu.Unlock()
		if item != nil {
			item.Remove()
		}
	}

	var appMu sync.Mutex
	var activeApp *app.Application

	var cleanupOnce sync.Once
	cleanup := func(reason string) {
		cleanupOnce.Do(func() {
			removeStatusItem()
			if logger != nil {
				logger.Info("shutdown", "reason", reason)
			}
			if model != nil {
				model.Shutdown()
			}
			if ipcSrv != nil {
				_ = ipcSrv.Close()
			}
			if transport != nil {
				_ = transport.Close()
			}
			if logger != nil {
				_ = logger.Close()
			}
		})
	}
	cleanupWithTimeout := func(reason string, timeout time.Duration) {
		done := make(chan struct{})
		go func() {
			cleanup(reason)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(timeout):
		}
	}

	var failsafeOnce sync.Once
	scheduleFailsafeExit := func(d time.Duration) {
		failsafeOnce.Do(func() {
			time.AfterFunc(d, func() {
				os.Exit(0)
			})
		})
	}

	requestExit := func(reason string, allow func()) {
		removeStatusItem()
		cleanupWithTimeout(reason, 800*time.Millisecond)
		scheduleFailsafeExit(1200 * time.Millisecond)
		if allow != nil {
			allow()
			return
		}
		appMu.Lock()
		a := activeApp
		appMu.Unlock()
		if a != nil {
			a.Quit()
			return
		}
		os.Exit(0)
	}

	defer cleanup("exit")

	model.OnQuit = func() {
		requestExit("quit_shortcut", nil)
	}

	// React cleanly to SIGINT / SIGTERM: remove tray icon, flush cache, remove IPC descriptor, close RPC server, and exit 0.
	go func() {
		<-ctx.Done()
		requestExit("signal", nil)
	}()

	page := ui.NewPage(ui.PageConfig{
		View:  model.Page,
		Theme: &model.Theme,
	})
	winIcon, _ := iconpng.Decode(assets.AppIconPNG)
	trayIcon, trayIconErr := iconpng.DecodeFit(assets.AppIconPNG, 64)
	cfg := ui.Configure(page, app.Config{
		Title:       "Parla Viva",
		Icon:        winIcon,
		Width:       1200,
		Height:      780,
		Web:         webHost,
		Invalidator: inv,
		OnWake:      model.DrainUpdates,
		OnFileDrop:  model.HandleFileDrop,
		OnActivate:  model.WakeNetwork,
		OnClosed: func() {
			requestExit("window_closed", nil)
		},
	})

	err := app.RunApplication(func(application *app.Application) error {
		appMu.Lock()
		activeApp = application
		appMu.Unlock()

		application.OnQuitRequest(func(allow func()) {
			requestExit("quit_request", allow)
		})

		win, err := application.OpenWindow(cfg, new(gfx.Canvas))
		if err != nil {
			return err
		}

		if trayIconErr == nil && trayIcon.Size > 0 {
			item, serr := application.AddStatusItem(app.StatusItemConfig{
				Tooltip: "Parla Viva",
				Icon: app.StatusIcon{
					Size: trayIcon.Size,
					RGBA: trayIcon.RGBA,
				},
				OnClick: func(_ app.StatusClick) {
					win.Show()
				},
				Menu: func() []app.MenuItem {
					return []app.MenuItem{
						{
							ID:    "show",
							Title: "Show Parla Viva",
							OnSelect: func() {
								win.Show()
							},
						},
						{Separator: true},
						{
							ID:    "quit",
							Title: "Quit Parla Viva",
							OnSelect: func() {
								requestExit("tray_quit", nil)
							},
						},
					}
				},
			})
			if serr == nil {
				statusMu.Lock()
				statusItem = item
				statusMu.Unlock()
			}
		}

		return nil
	})
	cleanup("exit")
	if err != nil {
		log.Fatal(err)
	}
}
