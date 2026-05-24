package page

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewStore_ディレクトリ自動作成 は、存在しないディレクトリを指定した場合に
// NewStore が自動的にディレクトリを作成することを検証する。
func TestNewStore_AutoCreateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newdir")

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore でエラーが発生: %v", err)
	}
	if store == nil {
		t.Fatal("store が nil")
	}

	// ディレクトリが実際に作成されているか確認する
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("ディレクトリが作成されていない: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("パスがディレクトリではない")
	}
}

// TestSaveAndGet は、ページの保存と取得が正しく動作することを検証する。
// フロントマター（title）が正しくYAMLとして書き込まれ、
// 読み込み時に正確にデコードされることを確認する。
func TestSaveAndGet(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	original := &Page{
		ID:    "hello-world",
		Title: "こんにちは世界",
		Body:  "# 見出し\n\nこれはテスト本文です。",
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save でエラー: %v", err)
	}

	got, err := store.Get("hello-world")
	if err != nil {
		t.Fatalf("Get でエラー: %v", err)
	}

	if got.ID != "hello-world" {
		t.Errorf("ID が不一致: got=%q, want=%q", got.ID, "hello-world")
	}
	if got.Title != "こんにちは世界" {
		t.Errorf("Title が不一致: got=%q, want=%q", got.Title, "こんにちは世界")
	}
	if got.Body != "# 見出し\n\nこれはテスト本文です。" {
		t.Errorf("Body が不一致: got=%q", got.Body)
	}
}

// TestList_更新日時降順 は、List() が更新日時の降順（新しい順）で返すことを検証する。
func TestList_OrderByUpdatedDesc(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	// 2つのページを順番に保存する（ファイルの更新日時が異なるようにする）
	// 高速な環境ではファイル更新日時が同一になることがあるため、
	// 明示的にsleepを入れて日時の差を確保する。
	store.Save(&Page{ID: "first", Title: "最初", Body: "1番目"})
	time.Sleep(50 * time.Millisecond)
	store.Save(&Page{ID: "second", Title: "次", Body: "2番目"})

	pages, err := store.List()
	if err != nil {
		t.Fatalf("List でエラー: %v", err)
	}

	if len(pages) != 2 {
		t.Fatalf("ページ数が不一致: got=%d, want=2", len(pages))
	}

	// second が後に保存されたので先頭に来るべき
	if pages[0].ID != "second" {
		t.Errorf("降順ソートが正しくない: 先頭が %q（'second' であるべき）", pages[0].ID)
	}
}

// TestDelete は、ページの削除が正しく動作することを検証する。
// 削除後に Get() でエラーが返ることを確認する。
func TestDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	store.Save(&Page{ID: "todelete", Title: "削除対象", Body: "test"})

	if err := store.Delete("todelete"); err != nil {
		t.Fatalf("Delete でエラー: %v", err)
	}

	_, err = store.Get("todelete")
	if err == nil {
		t.Error("削除したページが取得できてしまった")
	}
}

// TestDelete_履歴保存 は、ページ削除時に最終状態が履歴に保存されることを検証する。
func TestDelete_SaveHistory(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	store.Save(&Page{ID: "snap-target", Title: "削除テスト", Body: "最終状態の本文"})

	versions, err := store.History().ListVersions("snap-target")
	if err != nil {
		t.Fatalf("ListVersions でエラー: %v", err)
	}
	if len(versions) != 0 {
		t.Fatalf("削除前に履歴が存在する: %d件", len(versions))
	}

	if err := store.Delete("snap-target"); err != nil {
		t.Fatalf("Delete でエラー: %v", err)
	}

	versions, err = store.History().ListVersions("snap-target")
	if err != nil {
		t.Fatalf("ListVersions でエラー: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("履歴の件数が不一致: got=%d, want=1", len(versions))
	}

	histPage, err := store.History().LoadVersion("snap-target", versions[0].FileName)
	if err != nil {
		t.Fatalf("LoadVersion でエラー: %v", err)
	}
	if histPage.Title != "削除テスト" {
		t.Errorf("履歴のタイトルが不一致: got=%q, want=%q", histPage.Title, "削除テスト")
	}
	if histPage.Body != "最終状態の本文" {
		t.Errorf("履歴の本文が不一致: got=%q, want=%q", histPage.Body, "最終状態の本文")
	}

	_, err = store.Get("snap-target")
	if err == nil {
		t.Error("削除したページが取得できてしまった")
	}
}

// TestLoadLegacyFrontMatter は、tag/path 機能を廃止した後も、
// 旧フロントマター（tags: / path:）を含むファイルが
// エラーなく読み込めることを検証する（既存データ互換性）。
// goldmark/frontmatter は未知フィールドを自動で無視する。
func TestLoadLegacyFrontMatter(t *testing.T) {
	dir := t.TempDir()

	// 旧形式のフロントマターを持つMarkdownファイルを直接書き出す
	legacy := `---
title: "旧形式ページ"
tags:
  - "古いタグ"
path: "/old/path"
---

旧形式の本文。
`
	if err := os.WriteFile(filepath.Join(dir, "legacy-id.md"), []byte(legacy), 0644); err != nil {
		t.Fatalf("レガシーファイルの作成に失敗: %v", err)
	}

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore でエラー（旧データを読めない）: %v", err)
	}

	got, err := store.Get("legacy-id")
	if err != nil {
		t.Fatalf("旧形式ページの Get でエラー: %v", err)
	}
	if got.Title != "旧形式ページ" {
		t.Errorf("Title が不一致: got=%q", got.Title)
	}
	if got.Body == "" {
		t.Error("Body が空")
	}
}

// TestFrontMatterTimestamp_Created_Update は、frontmatter に created/update が
// 含まれるファイルを読み込んだ際に、ファイルシステムの ModTime ではなく
// frontmatter の値が CreatedAt/UpdatedAt として使われることを検証する。
func TestFrontMatterTimestamp_Created_Update(t *testing.T) {
	dir := t.TempDir()

	content := `---
title: "タイムスタンプテスト"
created: "2020-01-15T10:30:00+09:00"
update: "2024-06-20T14:00:00+09:00"
---

本文です。
`
	if err := os.WriteFile(filepath.Join(dir, "ts-test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("ファイルの作成に失敗: %v", err)
	}

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	got, err := store.Get("ts-test")
	if err != nil {
		t.Fatalf("Get でエラー: %v", err)
	}

	wantCreated := time.Date(2020, 1, 15, 10, 30, 0, 0, time.FixedZone("", 9*3600))
	wantUpdated := time.Date(2024, 6, 20, 14, 0, 0, 0, time.FixedZone("", 9*3600))

	if !got.CreatedAt.Equal(wantCreated) {
		t.Errorf("CreatedAt が不一致: got=%v, want=%v", got.CreatedAt, wantCreated)
	}
	if !got.UpdatedAt.Equal(wantUpdated) {
		t.Errorf("UpdatedAt が不一致: got=%v, want=%v", got.UpdatedAt, wantUpdated)
	}
}

// TestFrontMatterTimestamp_Fallback は、frontmatter に created/update がない
// 既存ファイルの場合にファイルの ModTime にフォールバックすることを検証する。
func TestFrontMatterTimestamp_Fallback(t *testing.T) {
	dir := t.TempDir()

	content := `---
title: "フォールバックテスト"
---

本文です。
`
	if err := os.WriteFile(filepath.Join(dir, "fb-test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("ファイルの作成に失敗: %v", err)
	}

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	got, err := store.Get("fb-test")
	if err != nil {
		t.Fatalf("Get でエラー: %v", err)
	}

	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt がゼロ値（ModTime フォールバックが動いていない）")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt がゼロ値（ModTime フォールバックが動いていない）")
	}
}

// TestSave_WritesTimestampToFrontMatter は、Save 後のファイルに
// created/update フィールドが正しく書き込まれることを検証する。
func TestSave_WritesTimestampToFrontMatter(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	now := time.Now()
	p := &Page{
		ID:        "save-ts",
		Title:     "保存テスト",
		Body:      "本文",
		CreatedAt: now.Add(-24 * time.Hour),
		UpdatedAt: now,
	}
	if err := store.Save(p); err != nil {
		t.Fatalf("Save でエラー: %v", err)
	}

	data, err := os.ReadFile(store.FilePath("save-ts"))
	if err != nil {
		t.Fatalf("ファイル読み込みに失敗: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "created:") {
		t.Error("保存されたファイルに created フィールドがない")
	}
	if !strings.Contains(content, "update:") {
		t.Error("保存されたファイルに update フィールドがない")
	}

	// 再読み込みしてタイムスタンプが保持されることを確認
	store2, err := NewStore(store.DataDir())
	if err != nil {
		t.Fatalf("再読み込みの NewStore でエラー: %v", err)
	}
	got, err := store2.Get("save-ts")
	if err != nil {
		t.Fatalf("再読み込み後の Get でエラー: %v", err)
	}

	if !got.CreatedAt.Equal(p.CreatedAt.Truncate(time.Second)) {
		t.Errorf("再読み込み後の CreatedAt が不一致: got=%v, want=%v", got.CreatedAt, p.CreatedAt)
	}
	if !got.UpdatedAt.Equal(p.UpdatedAt.Truncate(time.Second)) {
		t.Errorf("再読み込み後の UpdatedAt が不一致: got=%v, want=%v", got.UpdatedAt, p.UpdatedAt)
	}
}

// TestBuildGraph は、ストアからグラフ（ノード+エッジ）を構築する動作を検証する。
//   - 全ページがノードに含まれる（リンクのない孤立ノードも含む）
//   - WikiLink で結ばれたページ間にエッジが張られる
//   - 存在しないターゲットへのリンク（壊れたリンク）はエッジに含まれない
//   - 自己ループ（自分自身へのリンク）はエッジに含まれない
func TestBuildGraph(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore でエラー: %v", err)
	}

	// A → B, A → C (壊れた: D は存在しない), B → A, C → C (自己ループ), Z は孤立
	store.Save(&Page{ID: "A", Title: "ページA", Body: "[[B]] [[C]] [[D]]"})
	store.Save(&Page{ID: "B", Title: "ページB", Body: "[[A]]"})
	store.Save(&Page{ID: "C", Title: "ページC", Body: "[[C]]"})
	store.Save(&Page{ID: "Z", Title: "孤立Z", Body: "孤立しています"})

	nodes, edges := store.BuildGraph()

	// ノードは全4ページ（Z含む）
	if len(nodes) != 4 {
		t.Errorf("ノード数が不一致: got=%d, want=4, nodes=%v", len(nodes), nodes)
	}

	// 期待されるエッジ: A→B, A→C, B→A の3本。壊れた A→D と自己ループ C→C は含まない
	wantEdges := map[string]bool{
		"A->B": true,
		"A->C": true,
		"B->A": true,
	}
	if len(edges) != len(wantEdges) {
		t.Errorf("エッジ数が不一致: got=%d, want=%d, edges=%v", len(edges), len(wantEdges), edges)
	}
	for _, e := range edges {
		key := e.Source + "->" + e.Target
		if !wantEdges[key] {
			t.Errorf("予期しないエッジ: %s", key)
		}
	}
}
