package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Renderer はMarkdownテキストをHTMLに変換する。
// goldmarkライブラリを使用し、GFM拡張（テーブル・取り消し線・タスクリスト等）を有効にしている。
type Renderer struct {
	md goldmark.Markdown
}

// NewRenderer は新しいMarkdownレンダラを生成する。
// WikiLinkパーサーとCalloutトランスフォーマーも組み込まれている。
func NewRenderer() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&WikiLinkExtender{},
			&CalloutExtender{},
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)
	return &Renderer{md: md}
}

// Render はMarkdownテキストをHTMLに変換して返す。
// GFMテーブルの "|" とWikiLinkの "|" の衝突を回避するため、変換前後で
// [[...]] 内の "|" をプレースホルダに置換・復元する処理を行う。
func (r *Renderer) Render(source []byte) ([]byte, error) {
	source = preprocessWikiLinkPipes(source)
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return nil, err
	}
	return postprocessWikiLinkPipes(buf.Bytes()), nil
}
