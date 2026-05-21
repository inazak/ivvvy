package template

import "embed"

// TemplateFS はHTMLテンプレートファイルをバイナリに埋め込むための変数。
// Go 1.16以降の embed パッケージを使い、ビルド時にすべての .html ファイルを
// バイナリに含める。これによりシングルバイナリでの配布が可能になる。
//
//go:embed *.html
var TemplateFS embed.FS
