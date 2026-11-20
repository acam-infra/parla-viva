package main

import (
	"context"
	"flag"
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/appui"
	"github.com/acam-infra/parla-viva/internal/dc"
)

func main() {
	demoFlag := flag.Bool("demo", false, "run in offline preview mode without deltachat-rpc-server")
	rpcPath := flag.String("rpc", "", "explicit path to deltachat-rpc-server binary")
	accountsPath := flag.String("accounts", dc.DefaultAccountsDir(), "Delta Chat accounts directory")
	flag.Parse()

	inv := new(app.Invalidator)
	var model *appui.Model
	var transport *dc.Transport

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !*demoFlag {
		if bin := dc.DiscoverRPCServer(*rpcPath); bin != "" {
			transport = dc.NewTransport()
			if err := transport.StartProcess(bin, *accountsPath); err == nil {
				client := dc.NewClient(transport)
				model = appui.NewLiveModel(client, inv)
				model.StartLiveSync(ctx)
			} else {
				log.Printf("deltachat-rpc-server start failed (%v); falling back to demo mode", err)
			}
		}
	}

	if model == nil {
		model = appui.NewDemoModel()
		model.Invalidator = inv
	}
	if transport != nil {
		defer transport.Close()
	}

	page := ui.NewPage(ui.PageConfig{
		View:  model.Page,
		Theme: &model.Theme,
	})
	cfg := ui.Configure(page, app.Config{
		Title:       "Parla Viva",
		Width:       1140,
		Height:      760,
		Invalidator: inv,
		OnWake:      model.DrainUpdates,
	})

	if err := app.RunCanvas(cfg, new(gfx.Canvas)); err != nil {
		log.Fatal(err)
	}
}
