package www

import "embed"

//go:embed *.html
//go:embed *.js
//go:embed *.png
var Static embed.FS
