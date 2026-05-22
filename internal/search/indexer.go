package search

import (
	"encoding/json"

	"github.com/inazak/ivvvy/internal/page"
)

// IndexEntry はクライアントサイド検索用のインデックスエントリ。
// JSONに変換してブラウザに配信する。
type IndexEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// BuildIndex は全ページから検索用インデックスJSONを生成する。
func BuildIndex(pages []*page.Page) ([]byte, error) {
	entries := make([]IndexEntry, 0, len(pages))

	for _, p := range pages {
		entries = append(entries, IndexEntry{
			ID:    p.ID,
			Title: p.Title,
			Body:  p.Body,
		})
	}

	return json.Marshal(entries)
}
