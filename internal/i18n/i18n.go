// i18n パッケージはビルド時に選択された言語のUI文字列を提供する。
// デフォルトは英語。-tags lang_ja で日本語に切り替わる。
package i18n

// Messages はUI表示に使う翻訳済み文字列の集合。
type Messages struct {
	// ページタイトル（<title>タグ用）
	AllPages      string
	NewPage       string
	EditSuffix    string
	HistorySuffix string
	TagList       string
	TagPrefix     string
	GraphView     string

	// HTTPエラーメッセージ（ユーザーに表示される）
	ErrPageListFailed    string
	ErrMarkdownFailed    string
	ErrPageNotFound      string
	ErrHistoryListFailed string
	ErrHistoryNotFound   string
	ErrIndexFailed       string
	ErrRequestParse      string
	ErrTitleRequired     string
	ErrInvalidID         string
	ErrDuplicateID       string
	ErrSaveFailed        string
	ErrDeleteFailed      string
	ErrInternal          string
}

// Get はビルド時に選択された言語の Messages を返す。
func Get() *Messages { return &msg }
