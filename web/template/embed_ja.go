//go:build lang_ja

package template

import (
	"embed"
	"io/fs"
)

//go:embed ja/*.html
var rawFS embed.FS

func init() {
	sub, err := fs.Sub(rawFS, "ja")
	if err != nil {
		panic(err)
	}
	TemplateFS = sub
}
