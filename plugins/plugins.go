package plugins

import "embed"

//go:embed *.json apache caddy mariadb nginx php pip redis web
var BuiltIn embed.FS
