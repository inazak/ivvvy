package search

import (
	"encoding/json"
	"testing"

	"github.com/inazak/ivvvy/internal/page"
)

// TestBuildIndex は、ページ一覧から検索用JSONインデックスが正しく生成されることを検証する。
func TestBuildIndex(t *testing.T) {
	pages := []*page.Page{
		{
			ID:    "page1",
			Title: "テストドキュメント",
			Body:  "これはテストの本文です。",
		},
		{
			ID:    "page2",
			Title: "カードリーダ管理",
			Body:  "会員カード読み取り機の管理台帳。",
		},
	}

	data, err := BuildIndex(pages)
	if err != nil {
		t.Fatalf("BuildIndex でエラー: %v", err)
	}

	var entries []IndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("JSONのデコードに失敗: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("エントリ数が不一致: got=%d, want=2", len(entries))
	}

	e := entries[0]
	if e.ID != "page1" {
		t.Errorf("ID が不一致: got=%q", e.ID)
	}
	if e.Title != "テストドキュメント" {
		t.Errorf("Title が不一致: got=%q", e.Title)
	}
	if e.Body != "これはテストの本文です。" {
		t.Errorf("Body が不一致: got=%q", e.Body)
	}

	// tokens フィールドが JSON に含まれないことを確認する
	var raw []map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, ok := raw[0]["tokens"]; ok {
		t.Error("JSON に tokens フィールドが含まれている")
	}
}
