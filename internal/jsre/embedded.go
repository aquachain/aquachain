package jsre

import "embed"

//go:embed deps/*.js
var embedded embed.FS
