package caddypocketbase

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/caddy/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/sync/errgroup"
)

func init() {
	caddy.RegisterModule(new(App))
}

type pb interface {
	handler() http.Handler
}

// App is a Caddy module that provides an embedded PocketBase server.
//
// The module provides admin API endpoints under `/pocketbase/`:
//
// - `POST /pocketbase/superuser` - Create a new superuser
// - `PUT /pocketbase/superuser` - Upsert a superuser
// - `PATCH /pocketbase/superuser` - Update superuser password
// - `DELETE /pocketbase/superuser` - Delete a superuser
// - `POST /pocketbase/superuser/{email}/otp` - Generate OTP for superuser
//
// All the above endpoints require a JSON payload, except for the OTP endpoint. The
// JSON payload for the superuser endpoints is as follows:
//
//	{
//		"email_address": "...",
//		"password": "..."
//	}
//
// The `DELETE` endpoint does not expect the `password` field.
//
// Although PocketBase prints a URL in the logs to create the first superuser, the host
// part of the URL is not correct. You can either replace the host part with the host defined in
// your Caddy configuration, or use the admin API endpoint to create the first superuser.
//
// The app can be configured in the Caddyfile through the `pocketbase` block in the global options section. Syntax:
//
//	pocketbase {
//	    data_dir <path>
//	    listen   <addr>
//	    origins  <origin...>
//	}
//
// If the block is omitted, the default values are used.
type App struct {
	// The listen address of the PocketBase server. If empty, a free port on
	// 127.0.0.1 will be used.
	Listen string `json:"listen,omitempty"`

	// The data directory of PocketBase. If empty, a directory named `pb_data` in
	// the Caddy data directory will be used. Refer to [Caddy data directory](https://caddyserver.com/docs/conventions#data-directory)
	// for more information.
	DataDir string `json:"data_dir,omitempty"`

	// The allowed origins for the PocketBase server with respect to CORS.
	// If empty, all origins are allowed.
	Origins []string `json:"origins,omitempty"`

	ctx      caddy.Context
	pb       *pocketbase.PocketBase
	pbServer *http.Server
	errGroup *errgroup.Group
}

// CaddyModule implements caddy.Module.
func (a *App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pocketbase",
		New: func() caddy.Module {
			return new(App)
		},
	}
}

func (a *App) handler() http.Handler {
	return a.pbServer.Handler
}

// Provision sets up the needs of the PocketBase server.
// If `data_dir` is not set, a directory named `pb_data` is created in the Caddy data
// directory. If `listen` is not set, a free port on 127.0.0.1 is used.
// The PocketBase app is bootstrapped.
func (a *App) Provision(ctx caddy.Context) error {
	a.ctx = ctx

	if a.DataDir == "" {
		a.DataDir = filepath.Join(caddy.DefaultStorage.Path, "pb_data")
		if err := os.MkdirAll(a.DataDir, 0o7044); err != nil {
			return fmt.Errorf("not able to create data_dir: %w", err)
		}
	}

	if a.Listen == "" {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("not able to fine free port: %w", err)
		}
		a.Listen = l.Addr().String()
		l.Close()
	}

	a.pb = pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: a.DataDir,
	})

	if err := a.pb.Bootstrap(); err != nil {
		return err
	}
	a.pb.OnServe().BindFunc(func(e *core.ServeEvent) error {
		a.pbServer = e.Server
		return e.Next()
	})
	return nil
}

// Start starts the PocketBase server in a goroutine
func (a *App) Start() error {
	a.errGroup = new(errgroup.Group)
	a.errGroup.Go(func() error {
		e := apis.Serve(a.pb, apis.ServeConfig{
			HttpAddr:        a.Listen,
			ShowStartBanner: true,
			AllowedOrigins:  a.Origins,
		})
		return e
	})
	return nil
}

// Stop implements caddy.App.
func (a *App) Stop() error {
	err := a.pbServer.Shutdown(a.ctx)
	err = errors.Join(err, a.errGroup.Wait())
	event := new(core.TerminateEvent)
	event.App = a.pb
	return errors.Join(err, a.pb.OnTerminate().Trigger(event, func(e *core.TerminateEvent) error {
		return e.App.ResetBootstrapState()
	}))
}

var (
	_ caddy.Module      = (*App)(nil)
	_ caddy.Provisioner = (*App)(nil)
	_ caddy.App         = (*App)(nil)
	_ pb                = (*App)(nil)
)
