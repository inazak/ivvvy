package template

import "io/fs"

// TemplateFS はHTMLテンプレートファイルをバイナリに埋め込むための変数。
// ビルドタグにより en/ または ja/ のテンプレートが選択される。
// デフォルトは英語、-tags lang_ja で日本語になる。
var TemplateFS fs.FS
