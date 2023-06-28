package plugins

import "embed"

//go:embed *.json */*
var BuiltIn embed.FS
