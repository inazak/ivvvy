package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Callout はObsidian/Quartz形式のCalloutブロックを表すASTノード。
// blockquoteの先頭行が [!type] パターンにマッチする場合に使われる。
// HTML変換時に <div class="callout callout-{type}"> 形式に変換される。
type Callout struct {
	ast.BaseBlock
	// CalloutType は callout の種類（note, tip, warning, danger等）。CSSクラス名に使われる。
	CalloutType string

	// Title は callout のタイトル。
	// [!type] の後にテキストがある場合はそれが使われ、
	// ない場合は CalloutType に対応するデフォルトタイトルになる。
	Title string
}

// KindCallout はCalloutノードの種別を表す定数。
var KindCallout = ast.NewNodeKind("Callout")

// Kind はこのノードの種別を返す。
func (n *Callout) Kind() ast.NodeKind {
	return KindCallout
}

// Dump はデバッグ用のノード情報出力。
func (n *Callout) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// calloutPattern はblockquoteの先頭テキストから [!type] パターンを検出する正規表現。
var calloutPattern = regexp.MustCompile(`^\[!(\w+)\]\s*(.*)$`)

// calloutIcons はcalloutタイプごとのSVGアイコンを定義するマップ。
// stroke="currentColor" により、CSSの color プロパティに追従して色が変わる。
var calloutIcons = map[string]string{
	"note":     `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>`,
	"tip":      `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z"/></svg>`,
	"warning":  `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`,
	"danger":   `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>`,
	"info":     `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`,
	"example":  `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>`,
	"quote":    `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21z"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3z"/></svg>`,
	"abstract": `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><rect x="8" y="2" width="8" height="4" rx="1" ry="1"/></svg>`,
	"todo":     `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`,
	"bug":      `<svg class="callout-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="6" width="8" height="14" rx="4"/><path d="M19 10h2"/><path d="M3 10h2"/><path d="M19 14h2"/><path d="M3 14h2"/><path d="M19 18h2"/><path d="M3 18h2"/><path d="M12 2v4"/><path d="m9 3 3 3 3-3"/></svg>`,
}

// calloutDefaultTitles はcalloutタイプごとのデフォルトタイトル。
// [!type] の後にテキストが指定されていない場合に使われる。
var calloutDefaultTitles = map[string]string{
	"note":     "Note",
	"tip":      "Tip",
	"warning":  "Warning",
	"danger":   "Danger",
	"info":     "Info",
	"example":  "Example",
	"quote":    "Quote",
	"abstract": "Abstract",
	"todo":     "Todo",
	"bug":      "Bug",
}

// CalloutTransformer はgoldmarkのASTを走査し、
// blockquoteの先頭テキストが [!type] パターンにマッチする場合に
// CalloutノードにReplaceする変換器。
type CalloutTransformer struct{}

// Transform はドキュメント全体のASTを走査し、callout形式のblockquoteを変換する。
//
// ast.Walk 中に ReplaceChild でノードをツリーから外すと、外したノードの NextSibling
// が nil になり、Walk が後続兄弟へ進めなくなる（最初の callout だけ変換され、
// それ以降の blockquote が変換されないバグになる）。そのため、まず Walk で
// 対象 blockquote をすべて収集してから、収集後にまとめて変換する。
func (t *CalloutTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()

	var blockquotes []*ast.Blockquote
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if bq, ok := n.(*ast.Blockquote); ok {
			blockquotes = append(blockquotes, bq)
		}
		return ast.WalkContinue, nil
	})

	for _, bq := range blockquotes {
		firstText := extractFirstLineText(bq, source)
		if firstText == "" {
			continue
		}

		matches := calloutPattern.FindStringSubmatch(firstText)
		if matches == nil {
			continue
		}

		calloutType := strings.ToLower(matches[1])
		customTitle := strings.TrimSpace(matches[2])

		defaultTitle, known := calloutDefaultTitles[calloutType]
		if !known {
			continue
		}

		title := defaultTitle
		if customTitle != "" {
			title = customTitle
		}

		callout := &Callout{
			CalloutType: calloutType,
			Title:       title,
		}

		removeFirstTextLine(bq, source)

		for child := bq.FirstChild(); child != nil; {
			next := child.NextSibling()
			callout.AppendChild(callout, child)
			child = next
		}

		parent := bq.Parent()
		if parent == nil {
			// 既に外れている（外側のblockquoteが先にCallout化された等）はスキップ
			continue
		}
		parent.ReplaceChild(parent, bq, callout)
	}
}

// extractFirstLineText はblockquoteノードの最初の行のテキスト内容を文字列として取得する。
// goldmarkは "[!note]" を "[", "!note", "]" のように複数のTextノードに分割することがあるため、
// 最初のSoftLineBreak（改行）までの全Textノードを結合して返す。
func extractFirstLineText(bq *ast.Blockquote, source []byte) string {
	firstChild := bq.FirstChild()
	if firstChild == nil {
		return ""
	}

	var sb strings.Builder
	for child := firstChild.FirstChild(); child != nil; child = child.NextSibling() {
		tn, ok := child.(*ast.Text)
		if !ok {
			break
		}
		sb.Write(tn.Segment.Value(source))
		if tn.SoftLineBreak() {
			break
		}
	}
	return sb.String()
}

// removeFirstTextLine はblockquoteの先頭テキスト行（[!type] 行）を除去する。
// goldmarkは "[!note]" を複数のTextノードに分割することがあるため、
// 最初のSoftLineBreak（改行）を持つTextノードまでの全ノードを除去する。
// Paragraphが空になった場合はParagraphごと除去する。
func removeFirstTextLine(bq *ast.Blockquote, source []byte) {
	firstChild := bq.FirstChild()
	if firstChild == nil {
		return
	}

	for {
		child := firstChild.FirstChild()
		if child == nil {
			break
		}
		tn, ok := child.(*ast.Text)
		firstChild.RemoveChild(firstChild, child)
		if !ok {
			break
		}
		if tn.SoftLineBreak() {
			break
		}
	}

	if firstChild.FirstChild() == nil {
		bq.RemoveChild(bq, firstChild)
	}
}

// calloutHTMLRenderer はCalloutノードをHTMLの <div> 要素に変換するレンダラ。
type calloutHTMLRenderer struct{}

// RegisterFuncs はレンダラ関数をgoldmarkに登録する。
func (r *calloutHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindCallout, r.renderCallout)
}

// renderCallout はCalloutノードをHTMLに変換する。
// 以下のHTML構造を生成する:
//
//	<div class="callout callout-{type}">
//	  <div class="callout-title">{SVGアイコン}<span>{タイトル}</span></div>
//	  <div class="callout-content">{内容}</div>
//	</div>
func (r *calloutHTMLRenderer) renderCallout(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*Callout)
	if entering {
		w.WriteString(fmt.Sprintf(`<div class="callout callout-%s">`, n.CalloutType))
		w.WriteByte('\n')
		w.WriteString(`<div class="callout-title">`)
		if icon, ok := calloutIcons[n.CalloutType]; ok {
			w.WriteString(icon)
		}
		w.WriteString(fmt.Sprintf(`<span>%s</span>`, n.Title))
		w.WriteString("</div>\n")
		w.WriteString(`<div class="callout-content">`)
		w.WriteByte('\n')
	} else {
		w.WriteString("</div>\n")
		w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

// CalloutExtender はgoldmarkにCallout機能を追加する拡張。
type CalloutExtender struct{}

// Extend はgoldmarkにCalloutのASTTransformerとNodeRendererを登録する。
func (e *CalloutExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&CalloutTransformer{}, 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&calloutHTMLRenderer{}, 100),
		),
	)
}
