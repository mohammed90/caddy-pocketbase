package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// plug in Caddy modules here
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	_ "github.com/mohammed90/caddy-pocketbase"
)

func main() {
	caddycmd.Main()
}
