package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/inazak/ivvvy/internal/page"
)

// TagCount はタグ一覧表示用のタグ名とページ数のペア。
type TagCount struct {
	Name  string
	Count int
}

// TemplateData はHTMLテンプレートに渡すデータの共通構造体。
type TemplateData struct {
	Title string

	// Page は単一ページの情報。ページ閲覧・編集画面で使う。
	Page *page.Page

	// Pages はページの一覧。一覧画面で使う。
	Pages []*page.Page

	// Content はMarkdownから変換されたHTML。閲覧画面で使う。
	Content string

	// BacklinkPages はこのページへの WikiLink を含むページの一覧。
	BacklinkPages []*page.Page

	// RawBody はMarkdownの生テキスト。編集画面で使う。
	RawBody string

	// IsNew は新規作成かどうかを示すフラグ。
	IsNew bool

	// IsStatic は静的サイト生成モードかどうかを示すフラグ。
	// true の場合、編集ボタン・新規作成リンクをテンプレートで非表示にする。
	IsStatic bool

	// Versions はページの過去バージョン一覧。履歴一覧画面で使う。
	Versions []page.VersionEntry

	// IsVersionView は過去バージョン表示中かどうかを示すフラグ。
	// true の場合、view.html に過去バージョンのバナーを表示し、
	// 編集ボタン・履歴リンクを非表示にする。
	IsVersionView bool

	// VersionTimestamp は表示中の過去バージョンの時刻。
	// IsVersionView が true の場合のみ有効。
	VersionTimestamp time.Time

	// Tag はタグ絞り込みページで表示中のタグ名。
	Tag string

	// TagCounts はタグ一覧画面で表示するタグ名とページ数のリスト。
	TagCounts []TagCount
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", TemplateData{
		Title: "ivvvy",
	})
}

func (s *Server) handlePageList(w http.ResponseWriter, r *http.Request) {
	pages, err := s.store.List()
	if err != nil {
		http.Error(w, "ページ一覧の取得に失敗しました", http.StatusInternalServerError)
		return
	}

	s.render(w, "list.html", TemplateData{
		Title: "全ページ一覧",
		Pages: pages,
	})
}

// handlePageView は個別ページの閲覧画面を表示する。
// ページが存在しない場合は、IDをキーワードとして新規作成画面にリダイレクトする。
func (s *Server) handlePageView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := s.store.Get(id)
	if err != nil {
		// IDをキーワード（タイトル候補）として新規作成画面にリダイレクトする。
		// [[キーワード]] のWikiLinkから遷移した場合に、
		// キーワードがタイトルに入った状態で新規作成画面が開く。
		http.Redirect(w, r, "/new?title="+url.QueryEscape(id), http.StatusFound)
		return
	}

	html, err := s.renderer.Render([]byte(p.Body))
	if err != nil {
		http.Error(w, "Markdownの変換に失敗しました", http.StatusInternalServerError)
		return
	}

	backlinks, err := s.store.GetBacklinks(id)
	if err != nil {
		log.Printf("バックリンクの取得に失敗: %v", err)
	}

	s.render(w, "view.html", TemplateData{
		Title:         p.Title,
		Page:          p,
		Content:       string(html),
		BacklinkPages: backlinks,
	})
}

func (s *Server) handlePageEdit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := s.store.Get(id)
	if err != nil {
		http.Error(w, "ページが見つかりません", http.StatusNotFound)
		return
	}

	s.render(w, "edit.html", TemplateData{
		Title:   p.Title + " - 編集",
		Page:    p,
		RawBody: p.Body,
		IsNew:   false,
	})
}

// handlePageNew は新規ページの作成画面を表示する。
// クエリパラメータ title が指定されている場合はタイトル欄にプリセットする。
func (s *Server) handlePageNew(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("title")

	s.render(w, "edit.html", TemplateData{
		Title: "新規ページ作成",
		Page:  &page.Page{Title: keyword},
		IsNew: true,
	})
}

// handlePageHistory はページの過去バージョン一覧を表示する。
// 履歴ファイルは {dataDir}/history/{ID}/{TIMESTAMP}.md として保管されている。
func (s *Server) handlePageHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := s.store.Get(id)
	if err != nil {
		http.Error(w, "ページが見つかりません", http.StatusNotFound)
		return
	}

	versions, err := s.store.History().ListVersions(id)
	if err != nil {
		log.Printf("履歴一覧の取得に失敗: %v", err)
		http.Error(w, "履歴一覧の取得に失敗しました", http.StatusInternalServerError)
		return
	}

	s.render(w, "page_history.html", TemplateData{
		Title:    p.Title + " - 変更履歴",
		Page:     p,
		Versions: versions,
	})
}

// handlePageVersionView は指定された過去バージョンを表示する。
// filename は HistoryStore 側でディレクトリトラバーサル検証を行う。
func (s *Server) handlePageVersionView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	filename := chi.URLParam(r, "filename")

	p, err := s.store.History().LoadVersion(id, filename)
	if err != nil {
		// セキュリティ上、「不正なファイル名」「読み込み失敗」のどちらも 404 として返す。
		http.Error(w, "履歴ファイルが見つかりません", http.StatusNotFound)
		return
	}

	html, err := s.renderer.Render([]byte(p.Body))
	if err != nil {
		http.Error(w, "Markdownの変換に失敗しました", http.StatusInternalServerError)
		return
	}

	s.render(w, "view.html", TemplateData{
		Title:            p.Title,
		Page:             p,
		Content:          string(html),
		IsVersionView:    true,
		VersionTimestamp: p.UpdatedAt,
	})
}

// handleTagList はタグ一覧ページを表示する。
func (s *Server) handleTagList(w http.ResponseWriter, r *http.Request) {
	counts := s.store.TagCounts()

	tagCounts := make([]TagCount, 0, len(counts))
	for name, count := range counts {
		tagCounts = append(tagCounts, TagCount{Name: name, Count: count})
	}
	sort.Slice(tagCounts, func(i, j int) bool {
		return tagCounts[i].Name < tagCounts[j].Name
	})

	s.render(w, "tags.html", TemplateData{
		Title:     "タグ一覧",
		TagCounts: tagCounts,
	})
}

// handleTagView は指定されたタグを持つページの一覧を表示する。
// chi v5 はパスパラメータをパーセントデコードせず生の形のまま返すため、
// 日本語タグなどマルチバイト文字を含むタグでは明示的に PathUnescape する必要がある。
// （%XX の英字は大文字小文字どちらでも PathUnescape は同じ文字列にデコードする。）
func (s *Server) handleTagView(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if decoded, err := url.PathUnescape(tag); err == nil {
		tag = decoded
	}
	pages := s.store.ListByTag(tag)

	s.render(w, "tag_pages.html", TemplateData{
		Title: "タグ: " + tag,
		Tag:   tag,
		Pages: pages,
	})
}

func (s *Server) handleSearchIndex(w http.ResponseWriter, r *http.Request) {
	pages, err := s.store.List()
	if err != nil {
		http.Error(w, "インデックスの生成に失敗しました", http.StatusInternalServerError)
		return
	}

	data, err := s.indexer.BuildIndex(pages)
	if err != nil {
		http.Error(w, "インデックスの生成に失敗しました", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

// apiRequest はAPI経由でのページ作成・更新リクエストのJSON構造体。
type apiRequest struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Body  string   `json:"body"`
	Tags  []string `json:"tags"`
}

// handleAPIPageCreate は新規ページを作成するAPIエンドポイント。
// IDが省略された場合はミリ秒タイムスタンプで自動採番する。
func (s *Server) handleAPIPageCreate(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "リクエストの解析に失敗しました", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "タイトルは必須です", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		req.ID = fmt.Sprintf("%d", time.Now().UnixMilli())
	}

	// ディレクトリトラバーサル防止
	if strings.Contains(req.ID, "/") || strings.Contains(req.ID, "\\") || strings.Contains(req.ID, "..") {
		http.Error(w, "IDに不正な文字が含まれています", http.StatusBadRequest)
		return
	}

	if _, err := s.store.Get(req.ID); err == nil {
		http.Error(w, "同じIDのページがすでに存在します", http.StatusConflict)
		return
	}

	p := &page.Page{
		ID:        req.ID,
		Title:     req.Title,
		Body:      req.Body,
		Tags:      req.Tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.Save(p); err != nil {
		http.Error(w, "ページの保存に失敗しました", http.StatusInternalServerError)
		return
	}

	s.rewriteWikiLinks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": p.ID, "status": "created"})
}

// handleAPIPageUpdate は既存ページを更新するAPIエンドポイント。
func (s *Server) handleAPIPageUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "リクエストの解析に失敗しました", http.StatusBadRequest)
		return
	}

	existing, err := s.store.Get(id)
	if err != nil {
		http.Error(w, "ページが見つかりません", http.StatusNotFound)
		return
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	existing.Body = req.Body
	existing.Tags = req.Tags
	existing.UpdatedAt = time.Now()

	if err := s.store.Save(existing); err != nil {
		http.Error(w, "ページの保存に失敗しました", http.StatusInternalServerError)
		return
	}

	s.rewriteWikiLinks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "updated"})
}

// handleAPIPageDelete はページを削除するAPIエンドポイント。
func (s *Server) handleAPIPageDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := s.store.Get(id); err != nil {
		http.Error(w, "ページが見つかりません", http.StatusNotFound)
		return
	}

	if err := s.store.Delete(id); err != nil {
		http.Error(w, "ページの削除に失敗しました", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "deleted"})
}

func (s *Server) handleAPIPageRaw(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := s.store.Get(id)
	if err != nil {
		http.Error(w, "ページが見つかりません", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    p.ID,
		"title": p.Title,
		"body":  p.Body,
		"tags":  p.Tags,
	})
}

func (s *Server) handleAPIPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "リクエストの解析に失敗しました", http.StatusBadRequest)
		return
	}

	html, err := s.renderer.Render([]byte(req.Body))
	if err != nil {
		http.Error(w, "Markdownの変換に失敗しました", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"html": string(html)})
}

// handleGraphView はグラフビュー画面を表示する。
// 描画ライブラリ（cytoscape.js）はクライアント側で /api/graph.json を取得して描画する。
func (s *Server) handleGraphView(w http.ResponseWriter, r *http.Request) {
	s.render(w, "graph.html", TemplateData{
		Title: "グラフビュー",
	})
}

// handleAPIGraph は全ページをノード、WikiLinkをエッジとしたグラフJSONを返す。
// 孤立ノードも含めて全ページを nodes に入れる。
func (s *Server) handleAPIGraph(w http.ResponseWriter, r *http.Request) {
	nodes, edges := s.store.BuildGraph()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

// handleAPITags は全ページで使われているタグ名の一覧をJSON配列で返す。
func (s *Server) handleAPITags(w http.ResponseWriter, r *http.Request) {
	tags := s.store.ListAllTags()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(tags)
}

// wikiLinkKeywordPattern は [[キーワード]] パターン（"|" を含まない）にマッチする正規表現。
// [[ID|表示テキスト]] 形式はマッチしない。
var wikiLinkKeywordPattern = regexp.MustCompile(`\[\[([^\]|]+)\]\]`)

// rewriteWikiLinks は全ページの本文を走査し、
// [[キーワード]] をいずれかのページタイトルと照合して [[ID|キーワード]] に書き換える。
func (s *Server) rewriteWikiLinks() {
	pages, err := s.store.List()
	if err != nil {
		return
	}

	titleToID := make(map[string]string)
	for _, p := range pages {
		titleToID[p.Title] = p.ID
	}

	for _, p := range pages {
		newBody := wikiLinkKeywordPattern.ReplaceAllStringFunc(p.Body, func(match string) string {
			keyword := match[2 : len(match)-2]
			if id, ok := titleToID[keyword]; ok {
				if id == p.ID {
					return match
				}
				return "[[" + id + "|" + keyword + "]]"
			}
			return match
		})
		if newBody != p.Body {
			p.Body = newBody
			if err := s.store.Save(p); err != nil {
				log.Printf("WikiLink書き換え保存に失敗: %v", err)
			}
		}
	}
}

// render はHTMLテンプレートをレンダリングしてレスポンスに書き込む。
func (s *Server) render(w http.ResponseWriter, name string, data TemplateData) {
	tmpl, err := s.getTemplate(name)
	if err != nil {
		log.Printf("テンプレートの読み込みに失敗: %v", err)
		http.Error(w, "内部エラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("テンプレートのレンダリングに失敗: %v", err)
		http.Error(w, "内部エラー", http.StatusInternalServerError)
	}
}

// getTemplate は指定されたページテンプレートをキャッシュから取得する。
func (s *Server) getTemplate(name string) (*template.Template, error) {
	s.cacheMu.RLock()
	tmpl, ok := s.templateCache[name]
	s.cacheMu.RUnlock()
	if ok {
		return tmpl, nil
	}

	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(s.templateFS, "layout.html", name)
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.templateCache[name] = tmpl
	s.cacheMu.Unlock()

	return tmpl, nil
}
