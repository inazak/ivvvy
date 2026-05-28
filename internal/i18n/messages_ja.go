//go:build !lang_en

package i18n

var msg = Messages{
	AllPages:      "全ページ一覧",
	NewPage:       "新規ページ作成",
	EditSuffix:    " - 編集",
	HistorySuffix: " - 変更履歴",
	TagList:       "タグ一覧",
	TagPrefix:     "タグ: ",
	GraphView:     "グラフビュー",

	ErrPageListFailed:    "ページ一覧の取得に失敗しました",
	ErrMarkdownFailed:    "Markdownの変換に失敗しました",
	ErrPageNotFound:      "ページが見つかりません",
	ErrHistoryListFailed: "履歴一覧の取得に失敗しました",
	ErrHistoryNotFound:   "履歴ファイルが見つかりません",
	ErrIndexFailed:       "インデックスの生成に失敗しました",
	ErrRequestParse:      "リクエストの解析に失敗しました",
	ErrTitleRequired:     "タイトルは必須です",
	ErrInvalidID:         "IDに不正な文字が含まれています",
	ErrDuplicateID:       "同じIDのページがすでに存在します",
	ErrSaveFailed:        "ページの保存に失敗しました",
	ErrDeleteFailed:      "ページの削除に失敗しました",
	ErrInternal:          "内部エラー",
}
