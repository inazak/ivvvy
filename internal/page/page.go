package page

import (
	"time"
)

// Page はドキュメント管理の最小単位を表す構造体。
// すべてのドキュメントはMarkdownファイルとして保存され、
// YAMLフロントマターにタイトルを持つ。
type Page struct {
	// ID はファイル名から拡張子を除いたもの。ページの一意な識別子。
	ID string

	// Title はページのタイトル。フロントマターの title フィールドから取得する。
	Title string

	// Body はMarkdown本文（フロントマターを除いた部分）。
	Body string

	// Tags はページに付与されたタグ。フロントマターの tags フィールドから取得する。
	Tags []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// FrontMatter はMarkdownファイルの先頭にあるYAMLメタデータを表す構造体。
// goldmark/frontmatter パッケージでデコードする際に使う。
// 過去に存在した tags / path 等の未知キーは goldmark/frontmatter の
// デフォルト挙動で自動的に無視されるため、互換性に問題はない。
type FrontMatter struct {
	Title string   `yaml:"title"`
	Tags  []string `yaml:"tags"`
}
