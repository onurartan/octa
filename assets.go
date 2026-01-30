package octa

import "embed"

//go:embed web/*
var WebAssets embed.FS


//go:embed logo.png
var LogoData []byte 