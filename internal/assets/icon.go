package assets

import _ "embed"

// AppIconPNG is the 256x256 RGBA PNG application icon embedded into the binary.
//
//go:embed icon.png
var AppIconPNG []byte
