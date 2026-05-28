//go:build lang_en

package i18n

var msg = Messages{
	AllPages:      "All Pages",
	NewPage:       "New Page",
	EditSuffix:    " - Edit",
	HistorySuffix: " - History",
	TagList:       "Tags",
	TagPrefix:     "Tag: ",
	GraphView:     "Graph",

	ErrPageListFailed:    "Failed to retrieve page list",
	ErrMarkdownFailed:    "Markdown conversion failed",
	ErrPageNotFound:      "Page not found",
	ErrHistoryListFailed: "Failed to retrieve history",
	ErrHistoryNotFound:   "History file not found",
	ErrIndexFailed:       "Failed to generate index",
	ErrRequestParse:      "Failed to parse request",
	ErrTitleRequired:     "Title is required",
	ErrInvalidID:         "ID contains invalid characters",
	ErrDuplicateID:       "A page with this ID already exists",
	ErrSaveFailed:        "Failed to save page",
	ErrDeleteFailed:      "Failed to delete page",
	ErrInternal:          "Internal error",
}
