//go:build !lang_en

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
