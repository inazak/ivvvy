package static

import "embed"

// StaticFS はCSS/JavaScriptなどの静的ファイルをバイナリに埋め込むための変数。
// "all:" プレフィックスを付けることで、ドットで始まるファイルも含める。
// サブディレクトリ（css/, js/）配下のファイルもすべて埋め込まれる。
//
//go:embed all:css all:js
var StaticFS embed.FS
