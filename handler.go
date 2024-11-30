package caddypocketbase

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(new(Handler))
}

// Handler implements an HTTP handler that proxies requests internally to a PocketBase server.
// It can be used in conjunction with the PocketBase app. If the PocketBase app is not explicitly configured,
// a PocketBase app with default config is used.
type Handler struct {
	pbserver pb
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request, _ caddyhttp.Handler) error {
	h.pbserver.handler().ServeHTTP(rw, req)
	return nil
}

// Provision implements caddy.Provisioner.
func (h *Handler) Provision(ctx caddy.Context) error {
	a, err := ctx.App("pocketbase")
	if err != nil {
		return err
	}
	h.pbserver = a.(pb)
	return nil
}

// CaddyModule implements caddy.Module.
func (h *Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.pocketbase",
		New: func() caddy.Module {
			return new(Handler)
		},
	}
}

var (
	_ caddy.Module                = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
