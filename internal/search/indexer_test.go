package search

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/inazak/ivvvy/internal/page"
)

// TestNewIndexer は kagome形態素解析エンジンが正常に初期化できることを検証する。
func TestNewIndexer(t *testing.T) {
	idx, err := NewIndexer()
	if err != nil {
		t.Fatalf("NewIndexer でエラー: %v", err)
	}
	if idx == nil {
		t.Fatal("indexer が nil")
	}
}

// TestBuildIndex_基本動作 は、ページ一覧から検索用JSONインデックスが正しく生成されることを検証する。
func TestBuildIndex_BasicOperation(t *testing.T) {
	idx, err := NewIndexer()
	if err != nil {
		t.Fatalf("NewIndexer でエラー: %v", err)
	}

	pages := []*page.Page{
		{
			ID:    "page1",
			Title: "テストドキュメント",
			Body:  "これはテストの本文です。日本語の形態素解析が正しく動作するか確認します。",
		},
		{
			ID:    "page2",
			Title: "カードリーダ管理",
			Body:  "会員カード読み取り機の管理台帳。IPアドレスの割当を記録する。",
		},
	}

	data, err := idx.BuildIndex(pages)
	if err != nil {
		t.Fatalf("BuildIndex でエラー: %v", err)
	}

	// JSONとしてデコードできることを確認する
	var entries []IndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("JSONのデコードに失敗: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("エントリ数が不一致: got=%d, want=2", len(entries))
	}

	// 1件目のエントリの内容を検証する
	e := entries[0]
	if e.ID != "page1" {
		t.Errorf("ID が不一致: got=%q", e.ID)
	}
	if e.Title != "テストドキュメント" {
		t.Errorf("Title が不一致: got=%q", e.Title)
	}
}

// TestBuildIndex_トークン化 は、日本語の本文が形態素解析によって
// スペース区切りのトークン列に正しく変換されることを検証する。
// 助詞・助動詞が除外され、名詞や動詞が残っていることを確認する。
func TestBuildIndex_Tokenization(t *testing.T) {
	idx, err := NewIndexer()
	if err != nil {
		t.Fatalf("NewIndexer でエラー: %v", err)
	}

	pages := []*page.Page{
		{
			ID:    "tokentest",
			Title: "トークンテスト",
			Body:  "東京タワーは日本の観光名所です。",
		},
	}

	data, err := idx.BuildIndex(pages)
	if err != nil {
		t.Fatalf("BuildIndex でエラー: %v", err)
	}

	var entries []IndexEntry
	json.Unmarshal(data, &entries)

	tokens := entries[0].Tokens

	// 名詞「東京」「タワー」「日本」「観光」「名所」がトークンに含まれることを確認する
	for _, word := range []string{"東京", "タワー", "日本", "観光", "名所"} {
		if !strings.Contains(tokens, word) {
			t.Errorf("トークンに %q が含まれていない: tokens=%q", word, tokens)
		}
	}

	// 助詞「は」「の」がトークンに含まれないことを確認する（除外されているべき）
	// ただし他の単語の一部として含まれる可能性があるため、
	// スペース区切りの独立したトークンとして存在しないことを確認する
	tokenList := strings.Split(tokens, " ")
	for _, tok := range tokenList {
		if tok == "は" || tok == "の" {
			t.Errorf("助詞 %q がトークンに残っている", tok)
		}
	}
}
