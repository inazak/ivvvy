package markdown

import (
	"strings"
	"testing"
)

// TestRender_基本変換 は、基本的なMarkdownがHTMLに正しく変換されることを検証する。
func TestRender_BasicConversion(t *testing.T) {
	r := NewRenderer()

	input := []byte("# 見出し\n\nこれは段落です。\n\n- リスト項目1\n- リスト項目2\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// h1 タグが生成されていることを確認する
	if !strings.Contains(result, "<h1>見出し</h1>") {
		t.Errorf("h1 タグが見つからない: %s", result)
	}

	// 段落が p タグで囲まれていることを確認する
	if !strings.Contains(result, "<p>") {
		t.Errorf("p タグが見つからない: %s", result)
	}

	// リスト項目が li タグで囲まれていることを確認する
	if !strings.Contains(result, "<li>リスト項目1</li>") {
		t.Errorf("li タグが見つからない: %s", result)
	}
}

// TestRender_WikiLink は、[[ページ名]] 記法が <a> タグに正しく変換されることを検証する。
func TestRender_WikiLink(t *testing.T) {
	r := NewRenderer()

	input := []byte("[[my-page]] へのリンクです。")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// WikiLinkが <a> タグに変換されていることを確認する
	if !strings.Contains(result, `href="/page/my-page"`) {
		t.Errorf("WikiLink の href が見つからない: %s", result)
	}
	if !strings.Contains(result, `class="wikilink"`) {
		t.Errorf("WikiLink の class が見つからない: %s", result)
	}
	if !strings.Contains(result, ">my-page</a>") {
		t.Errorf("WikiLink のテキストが見つからない: %s", result)
	}
}

// TestRender_WikiLink_日本語 は、[[日本語ページ名]] が正しく変換されることを検証する。
func TestRender_WikiLink_Japanese(t *testing.T) {
	r := NewRenderer()

	input := []byte("[[テストページ]] を参照してください。")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)
	if !strings.Contains(result, `href="/page/テストページ"`) {
		t.Errorf("日本語WikiLink が変換されていない: %s", result)
	}
}

// TestRender_コードブロック は、コードブロックが <pre><code> に変換されることを検証する。
func TestRender_CodeBlock(t *testing.T) {
	r := NewRenderer()

	input := []byte("```\nfunc main() {}\n```\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)
	if !strings.Contains(result, "<pre>") || !strings.Contains(result, "<code>") {
		t.Errorf("コードブロックが正しく変換されていない: %s", result)
	}
}

// TestRender_WikiLink_パイプ記法 は、[[ID|表示テキスト]] 形式のWikiLinkが
// 正しく変換されることを検証する。IDをリンク先、表示テキストをリンクテキストとして分離する。
func TestRender_WikiLink_PipeSyntax(t *testing.T) {
	r := NewRenderer()

	input := []byte("[[1715644800000|ページタイトル]] を参照してください。")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// hrefにはIDが使われていることを確認する
	if !strings.Contains(result, `href="/page/1715644800000"`) {
		t.Errorf("WikiLink の href にIDが使われていない: %s", result)
	}

	// リンクテキストには表示テキストが使われていることを確認する
	if !strings.Contains(result, ">ページタイトル</a>") {
		t.Errorf("WikiLink のテキストが表示テキストになっていない: %s", result)
	}
}

// TestRender_Callout_note は、> [!note] 記法がCalloutブロックに変換されることを検証する。
func TestRender_Callout_note(t *testing.T) {
	r := NewRenderer()

	input := []byte("> [!note]\n> メモの内容です。\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// callout-note クラスのdivが生成されていることを確認する
	if !strings.Contains(result, `class="callout callout-note"`) {
		t.Errorf("callout-note クラスが見つからない: %s", result)
	}

	// タイトル "Note" が含まれていることを確認する
	if !strings.Contains(result, "Note") {
		t.Errorf("デフォルトタイトル 'Note' が見つからない: %s", result)
	}

	// SVGアイコンが含まれていることを確認する
	if !strings.Contains(result, `class="callout-icon"`) {
		t.Errorf("SVGアイコンが見つからない: %s", result)
	}

	// コンテンツが含まれていることを確認する
	if !strings.Contains(result, "メモの内容です。") {
		t.Errorf("コンテンツが見つからない: %s", result)
	}
}

// TestRender_Callout_カスタムタイトル は、> [!warning] タイトル 形式で
// カスタムタイトルが反映されることを検証する。
func TestRender_Callout_CustomTitle(t *testing.T) {
	r := NewRenderer()

	input := []byte("> [!warning] 注意してください\n> 警告の内容です。\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// callout-warning クラスのdivが生成されていることを確認する
	if !strings.Contains(result, `class="callout callout-warning"`) {
		t.Errorf("callout-warning クラスが見つからない: %s", result)
	}

	// カスタムタイトルが使われていることを確認する
	if !strings.Contains(result, "注意してください") {
		t.Errorf("カスタムタイトルが見つからない: %s", result)
	}
}

// TestRender_通常Blockquote は、[!type] パターンのない通常のblockquoteが
// Calloutに変換されずそのまま残ることを検証する。
func TestRender_NormalBlockquote(t *testing.T) {
	r := NewRenderer()

	input := []byte("> 通常の引用テキストです。\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// blockquoteとして残っていることを確認する
	if !strings.Contains(result, "<blockquote>") {
		t.Errorf("通常のblockquoteがblockquoteタグで出力されていない: %s", result)
	}

	// calloutクラスが付いていないことを確認する
	if strings.Contains(result, "callout") {
		t.Errorf("通常のblockquoteがcalloutに変換されてしまった: %s", result)
	}
}

// TestRender_Callout_複数連続 は、同一ドキュメント内に複数のCalloutブロックが
// 並んだ場合、すべてが正しく変換されることを検証する。
// ast.Walk 中に ReplaceChild でノードをツリーから外すと NextSibling が nil になり、
// 最初の callout だけ変換されて以降の blockquote が変換されない退行が発生したことの回帰テスト。
func TestRender_Callout_MultipleConsecutive(t *testing.T) {
	r := NewRenderer()

	input := []byte("> [!note]\n> a\n\n> [!warning]\n> b\n\n> [!tip]\n> c\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}
	result := string(html)

	for _, ct := range []string{"note", "warning", "tip"} {
		expected := `class="callout callout-` + ct + `"`
		if !strings.Contains(result, expected) {
			t.Errorf("callout-%s が変換されていない（blockquoteのまま）: %s", ct, result)
		}
	}
}

// TestRender_GFMテーブル は、GitHub Flavored Markdownのテーブルが
// <table> タグに変換されることを検証する。
// 管理台帳など表形式のデータを記述する際に必要。
func TestRender_GFMTable(t *testing.T) {
	r := NewRenderer()

	input := []byte("| 名前 | 値 |\n|------|----|\n| A | 1 |\n| B | 2 |\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)
	if !strings.Contains(result, "<table>") {
		t.Errorf("table タグが見つからない: %s", result)
	}
	if !strings.Contains(result, "<th>名前</th>") {
		t.Errorf("th タグが見つからない: %s", result)
	}
}

// TestRender_WikiLink_テーブルセル内 は、GFMテーブルのセル内に
// パイプ記法の WikiLink [[ID|表示テキスト]] を書いてもテーブルが崩れず、
// WikiLinkも正しくリンク化されることを検証する。
// テーブルのセル区切り "|" と WikiLink の表示テキスト区切り "|" の
// 衝突を解消するための回帰テスト。
func TestRender_WikiLink_InTableCell(t *testing.T) {
	r := NewRenderer()

	input := []byte("| 項目 | 参照 |\n|------|------|\n| A | [[1715644800000|ページタイトル]] |\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// テーブル構造が壊れていないこと: th が2つ、td が2つ
	if strings.Count(result, "<th>") != 2 {
		t.Errorf("th が2つでない（テーブルが崩れた可能性）: %s", result)
	}
	if strings.Count(result, "<td>") != 2 {
		t.Errorf("td が2つでない（テーブルが崩れた可能性）: %s", result)
	}

	// WikiLinkが正しくリンク化されていること
	if !strings.Contains(result, `href="/page/1715644800000"`) {
		t.Errorf("テーブルセル内のWikiLinkのhrefが見つからない: %s", result)
	}
	if !strings.Contains(result, ">ページタイトル</a>") {
		t.Errorf("テーブルセル内のWikiLinkの表示テキストが見つからない: %s", result)
	}

	// プレースホルダ文字がHTMLに残っていないこと
	if strings.ContainsRune(result, '\x1F') {
		t.Errorf("プレースホルダ \\x1F が出力に残っている: %q", result)
	}
}

// TestRender_WikiLink_コードブロック内パイプ は、フェンスドコードブロック内の
// [[a|b]] が WikiLink として解釈されず、リテラルテキストとして
// "|" がそのまま保持されることを検証する。
// 前処理で "|" をプレースホルダに置換しても、コードブロック内では
// 後処理で "|" に復元されるべき。
func TestRender_WikiLink_PipeInCodeBlock(t *testing.T) {
	r := NewRenderer()

	input := []byte("```\n[[a|b]]\n```\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// コードブロック中ではリンク化されていないこと
	if strings.Contains(result, `class="wikilink"`) {
		t.Errorf("コードブロック内の [[a|b]] がリンク化されてしまった: %s", result)
	}

	// パイプがリテラルとして保持されていること
	if !strings.Contains(result, "[[a|b]]") {
		t.Errorf("コードブロック内のリテラル [[a|b]] が残っていない: %s", result)
	}

	// プレースホルダが残っていないこと
	if strings.ContainsRune(result, '\x1F') {
		t.Errorf("プレースホルダ \\x1F が出力に残っている: %q", result)
	}
}

// TestRender_WikiLink_インラインコード内パイプ は、インラインコードスパン
// `[[a|b]]` 中の "|" がリテラルとして保持されることを検証する。
func TestRender_WikiLink_PipeInInlineCode(t *testing.T) {
	r := NewRenderer()

	input := []byte("インラインで `[[a|b]]` と書く。\n")
	html, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render でエラー: %v", err)
	}

	result := string(html)

	// コードスパン中ではリンク化されていないこと
	if strings.Contains(result, `class="wikilink"`) {
		t.Errorf("インラインコード内の [[a|b]] がリンク化されてしまった: %s", result)
	}
	if !strings.Contains(result, "[[a|b]]") {
		t.Errorf("インラインコード内のリテラル [[a|b]] が残っていない: %s", result)
	}
	if strings.ContainsRune(result, '\x1F') {
		t.Errorf("プレースホルダ \\x1F が出力に残っている: %q", result)
	}
}
