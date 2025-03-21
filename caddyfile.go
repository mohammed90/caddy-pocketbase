package caddypocketbase

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("pocketbase", parseCaddyfile)
	httpcaddyfile.RegisterGlobalOption("pocketbase", parseGlobalOption)
}

// parseCaddyfile parses the pocketbase directive. Syntax:
//
//	pocketbase [<matcher>]
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var handler Handler

	return &handler, nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

// parseGlobalOption parses the pocketbase directive. Syntax:
//
//	pocketbase {
//	    data_dir <path>
//	    listen   <addr>
//	    origins  <origin...>
//	}
func parseGlobalOption(d *caddyfile.Dispenser, existingVal any) (any, error) {
	app := new(App)
	d.Next()
	caddy.Log().Info("parseGlobalOption")
	for d.NextBlock(0) {
		caddy.Log().Info("in loop", zap.String("directive", d.Val()))
		switch d.Val() {
		case "data_dir":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			app.DataDir = d.Val()
		case "listen":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			app.Listen = d.Val()
		case "origins":
			app.Origins = append(app.Origins, d.Val())
		default:
			return nil, d.Errf("unrecognized subdirective '%s'", d.Val())
		}
	}
	return httpcaddyfile.App{
		Name:  "pocketbase",
		Value: caddyconfig.JSON(app, nil),
	}, nil
}
