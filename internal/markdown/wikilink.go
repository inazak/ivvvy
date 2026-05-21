package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// wikiLinkPipePlaceholder は [[ターゲット|表示テキスト]] の "|" を一時的に置き換える
// プレースホルダ文字。GFMテーブル記法の "|" との衝突を避けるため、Render前に
// [[...]] 内のパイプをこの文字に置換し、Render後にHTML中に残ったものを "|" に戻す。
// U+001F (Unit Separator) は制御文字で通常のテキストには現れないため安全に使える。
const wikiLinkPipePlaceholder byte = '\x1F'

// preprocessWikiLinkPipes はソース中の [[...]] パターン内の "|" を
// wikiLinkPipePlaceholder に置換する。閉じ括弧 ]] が同一行内に見つからない
// 場合は何もしない。これにより、GFMテーブルのセル区切り "|" と
// WikiLinkの表示テキスト区切り "|" の衝突を解消する。
//
// コードブロック（フェンス・インライン共通）の検出はあえて行わず、
// 後段の postprocessWikiLinkPipes で復元することで一貫した挙動を得る。
func preprocessWikiLinkPipes(source []byte) []byte {
	if len(source) < 5 {
		return source
	}
	out := make([]byte, 0, len(source))
	i := 0
	for i < len(source) {
		// "[[" を検出
		if i+1 < len(source) && source[i] == '[' && source[i+1] == '[' {
			// 同一行内で対応する "]]" を探す
			end := -1
			for j := i + 2; j < len(source)-1; j++ {
				if source[j] == '\n' {
					break
				}
				if source[j] == ']' && source[j+1] == ']' {
					end = j
					break
				}
			}
			if end == -1 {
				out = append(out, source[i])
				i++
				continue
			}
			// "[[" をそのまま書き出す
			out = append(out, source[i], source[i+1])
			// 中身の "|" をプレースホルダに置換
			for k := i + 2; k < end; k++ {
				if source[k] == '|' {
					out = append(out, wikiLinkPipePlaceholder)
				} else {
					out = append(out, source[k])
				}
			}
			// "]]" をそのまま書き出す
			out = append(out, source[end], source[end+1])
			i = end + 2
			continue
		}
		out = append(out, source[i])
		i++
	}
	return out
}

// postprocessWikiLinkPipes はHTML中に残った wikiLinkPipePlaceholder を
// "|" に戻す。WikiLinkとして解釈されたものはレンダラ側で既に "|" を含まない
// 形でHTML化されるため、ここで復元されるのはコードブロック/コード span/
// その他WikiLinkに解釈されなかった生テキスト中のプレースホルダのみ。
func postprocessWikiLinkPipes(html []byte) []byte {
	if !bytes.ContainsAny(html, "\x1F") {
		return html
	}
	return bytes.ReplaceAll(html, []byte{wikiLinkPipePlaceholder}, []byte("|"))
}

// WikiLink は [[ターゲット]] または [[ターゲット|表示テキスト]] 形式のリンクを表すASTノード。
type WikiLink struct {
	ast.BaseInline
	// Target はリンク先のページID（またはキーワード）。
	// [[ID|表示テキスト]] の場合は "|" の前の部分。
	Target string

	// DisplayText はリンクとして表示するテキスト。
	// [[ID|表示テキスト]] の場合は "|" の後の部分。
	// [[キーワード]] のように "|" がない場合は Target と同じ値になる。
	DisplayText string
}

// KindWikiLink はWikiLinkノードの種別を表す定数。
var KindWikiLink = ast.NewNodeKind("WikiLink")

// Kind はこのノードの種別を返す。
func (n *WikiLink) Kind() ast.NodeKind {
	return KindWikiLink
}

// Dump はデバッグ用のノード情報出力。
func (n *WikiLink) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// wikiLinkParser は Markdown テキスト中の [[...]] パターンを検出し、
// WikiLink ASTノードに変換するインラインパーサー。
type wikiLinkParser struct{}

// Trigger はこのパーサーが反応する開始文字を返す。
func (p *wikiLinkParser) Trigger() []byte {
	return []byte{'['}
}

// Parse はテキスト中の [[...]] パターンを解析し、WikiLinkノードを生成する。
// [[ターゲット]] → Target=ターゲット, DisplayText=ターゲット
// [[ID|表示テキスト]] → Target=ID, DisplayText=表示テキスト
func (p *wikiLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 5 || line[0] != '[' || line[1] != '[' {
		return nil
	}

	for i := 2; i < len(line)-1; i++ {
		if line[i] == ']' && line[i+1] == ']' {
			content := string(line[2:i])
			if content == "" {
				return nil
			}
			block.Advance(i + 2)

			target := content
			displayText := content
			// 通常は preprocessWikiLinkPipes により "|" がプレースホルダに置換されている。
			// プレースホルダがなければ "|" にもフォールバック（直接パーサを使う経路への保険）。
			if idx := strings.IndexByte(content, wikiLinkPipePlaceholder); idx >= 0 {
				target = content[:idx]
				displayText = content[idx+1:]
			} else if idx := strings.Index(content, "|"); idx >= 0 {
				target = content[:idx]
				displayText = content[idx+1:]
			}

			node := &WikiLink{Target: target, DisplayText: displayText}
			return node
		}
	}
	return nil
}

// wikiLinkHTMLRenderer は WikiLink ASTノードをHTMLの <a> タグに変換する。
type wikiLinkHTMLRenderer struct{}

// RegisterFuncs はレンダラ関数をgoldmarkに登録する。
func (r *wikiLinkHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindWikiLink, r.renderWikiLink)
}

// renderWikiLink は WikiLink ノードをHTMLに変換する。
// <a href="/page/ターゲット" class="wikilink">表示テキスト</a> を出力する。
func (r *wikiLinkHTMLRenderer) renderWikiLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*WikiLink)
		w.WriteString(`<a href="/page/`)
		w.WriteString(n.Target)
		w.WriteString(`" class="wikilink">`)
		w.WriteString(n.DisplayText)
	} else {
		w.WriteString(`</a>`)
	}
	return ast.WalkContinue, nil
}

// WikiLinkExtender はgoldmarkに WikiLink 機能を追加する拡張。
type WikiLinkExtender struct{}

// Extend はgoldmarkのMarkdownインスタンスにWikiLinkのパーサーとレンダラを登録する。
func (e *WikiLinkExtender) Extend(m goldmark.Markdown) {
	// 標準のリンクパーサー（優先度100）より先に [[...]] を検出するため優先度50で登録する。
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(&wikiLinkParser{}, 50),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&wikiLinkHTMLRenderer{}, 50),
		),
	)
}
