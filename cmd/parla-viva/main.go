package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/app/iconpng"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/agentcli"
	"github.com/acam-infra/parla-viva/internal/applog"
	"github.com/acam-infra/parla-viva/internal/config"
	"github.com/acam-infra/parla-viva/internal/appui"
	"github.com/acam-infra/parla-viva/internal/assets"
	"github.com/acam-infra/parla-viva/internal/dc"
)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	if ipcSrv != nil {
		defer ipcSrv.Close()
	}
	if transport != nil {
		defer transport.Close()
	}

	page := ui.NewPage(ui.PageConfig{
		View:  model.Page,
		Theme: &model.Theme,
	})
	winIcon, _ := iconpng.Decode(assets.AppIconPNG)
	cfg := ui.Configure(page, app.Config{
		Title:       "Parla Viva",
		Icon:        winIcon,
		Width:       1200,
		Height:      780,
		Web:         webHost,
		Invalidator: inv,
		OnWake:      model.DrainUpdates,
		OnFileDrop:  model.HandleFileDrop,
	})

	if err := app.RunCanvas(cfg, new(gfx.Canvas)); err != nil {
		log.Fatal(err)
	}
}
