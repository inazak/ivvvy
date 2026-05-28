//go:build !lang_ja

package template

import (
	"embed"
	"io/fs"
)

//go:embed en/*.html
var rawFS embed.FS

func init() {
	sub, err := fs.Sub(rawFS, "en")
	if err != nil {
		panic(err)
	}
	TemplateFS = sub
}
