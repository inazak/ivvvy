package search

import (
	"encoding/json"
	"strings"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"

	"github.com/inazak/ivvvy/internal/page"
)

// IndexEntry はクライアントサイド検索用のインデックスエントリ。
// JSONに変換してブラウザに配信する。
type IndexEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`

	// Tokens は本文を形態素解析して得られたトークン列（スペース区切り）。
	// 日本語テキストをスペース区切りに変換することで、
	// クライアントサイドのAND部分一致検索で使えるようにする。
	Tokens string `json:"tokens"`

	// Body は本文の原文。クライアントサイドでスニペットを切り出す際に使う。
	Body string `json:"body"`
}

// Indexer は kagome（形態素解析エンジン）を使って
// ページの本文をトークン化し、検索用インデックスを生成する。
type Indexer struct {
	tok *tokenizer.Tokenizer
}

// NewIndexer は新しいインデクサーを生成する。IPA辞書を内蔵した kagome トークナイザーを初期化する。
func NewIndexer() (*Indexer, error) {
	tok, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		return nil, err
	}
	return &Indexer{tok: tok}, nil
}

// BuildIndex は全ページから検索用インデックスJSONを生成する。
func (idx *Indexer) BuildIndex(pages []*page.Page) ([]byte, error) {
	entries := make([]IndexEntry, 0, len(pages))

	for _, p := range pages {
		tokens := idx.tokenize(p.Title + " " + p.Body)

		entries = append(entries, IndexEntry{
			ID:     p.ID,
			Title:  p.Title,
			Tokens: tokens,
			Body:   p.Body,
		})
	}

	return json.Marshal(entries)
}

// tokenize はテキストを形態素解析し、スペース区切りのトークン列に変換する。
// 助詞・助動詞・記号などの不要な品詞を除外し、検索に有用な単語のみを残す。
func (idx *Indexer) tokenize(text string) string {
	tokens := idx.tok.Tokenize(text)

	var parts []string
	for _, token := range tokens {
		features := token.Features()
		if len(features) == 0 {
			continue
		}

		pos := features[0]

		switch pos {
		case "助詞", "助動詞", "記号":
			continue
		}

		surface := token.Surface
		if surface != "" {
			parts = append(parts, surface)
		}
	}

	return strings.Join(parts, " ")
}
