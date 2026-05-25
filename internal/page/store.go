package page

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"go.abhg.dev/goldmark/frontmatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// wikiLinkPattern は本文中の WikiLink（[[ID]] または [[ID|表示テキスト]]）を
// 検出するための正規表現。バックリンクインデックスの構築に使用する。
var wikiLinkPattern = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// Store はフラットファイルシステム上のMarkdownファイルを読み書きする。
// データベースを使わず、指定されたディレクトリにすべてのページを保存する。
// 全ページをメモリ上にキャッシュし、読み取り時のディスクI/Oを排除している。
// すべての書き込みは Store のメソッドを経由するため、キャッシュとディスクの一貫性が保たれる。
type Store struct {
	dataDir string

	// ファイル書き込み・削除操作をシリアライズするためのミューテックス。
	mu sync.RWMutex

	// 全ページをメモリ上に保持するマップ。キーはページID、値はPageのポインタ。
	// 起動時に全ファイルを読み込み、Save/Deleteで同期的に更新する。
	cache map[string]*Page

	// バックリンクの逆引きインデックス。
	// キーはリンク先ページID、値はリンク元ページIDのスライス。
	// 起動時に全ページの本文から WikiLink を抽出して構築し、
	// Save/Delete 時に該当ページ分だけ差分更新する。
	backlinks map[string][]string

	history *HistoryStore
}

// NewStore は指定されたディレクトリをデータ保存先とする Store を生成する。
// ディレクトリが存在しない場合は自動的に作成する。
// 起動時に全ページをメモリに読み込むため、ページ数に比例した初期化時間がかかる。
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("データディレクトリの作成に失敗: %w", err)
	}

	hs, err := NewHistoryStore(filepath.Join(dataDir, "history"))
	if err != nil {
		return nil, fmt.Errorf("履歴ストアの初期化に失敗: %w", err)
	}

	s := &Store{
		dataDir:   dataDir,
		cache:     make(map[string]*Page),
		backlinks: make(map[string][]string),
		history:   hs,
	}

	if err := s.loadAllToCache(); err != nil {
		return nil, fmt.Errorf("ページキャッシュの初期化に失敗: %w", err)
	}

	s.buildBacklinksIndex()

	return s, nil
}

// History は版管理ストアへのアクセサ。
func (s *Store) History() *HistoryStore {
	return s.history
}

// loadAllToCache はデータディレクトリ内の全 .md ファイルを読み込み、
// キャッシュに格納する。起動時に一度だけ呼ばれる。
func (s *Store) loadAllToCache() error {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return fmt.Errorf("データディレクトリの読み込みに失敗: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".md")
		p, err := s.loadPageFromDisk(id)
		if err != nil {
			// パース失敗のファイルはスキップする（壊れたファイルがあっても起動できるように）
			continue
		}
		s.cache[id] = p
	}
	return nil
}

// Get は指定されたIDのページをキャッシュから取得する。
// キャッシュ汚染を防ぐためコピーを返す。
func (s *Store) Get(id string) (*Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.cache[id]
	if !ok {
		return nil, fmt.Errorf("ページが見つかりません: %s", id)
	}
	return s.copyPage(p), nil
}

// List はすべてのページを更新日時の降順で返す。
func (s *Store) List() ([]*Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pages := make([]*Page, 0, len(s.cache))
	for _, p := range s.cache {
		pages = append(pages, s.copyPage(p))
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].UpdatedAt.After(pages[j].UpdatedAt)
	})

	return pages, nil
}

// Save はページをMarkdownファイルとして保存し、キャッシュも更新する。
// 上書き直前に既存ファイルを HistoryStore へ退避する。
// 新規作成（既存ファイルなし）の場合は退避をスキップする。
func (s *Store) Save(page *Page) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dataDir, page.ID+".md")

	if oldBytes, rerr := os.ReadFile(filePath); rerr == nil && len(oldBytes) > 0 && s.history != nil {
		if serr := s.history.Snapshot(page.ID, oldBytes); serr != nil {
			_ = serr
		}
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: %q\n", page.Title))
	if len(page.Tags) > 0 {
		buf.WriteString("tags:\n")
		for _, tag := range page.Tags {
			buf.WriteString(fmt.Sprintf("  - %q\n", tag))
		}
	}
	buf.WriteString(fmt.Sprintf("created: %q\n", page.CreatedAt.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("update: %q\n", page.UpdatedAt.Format(time.RFC3339)))
	buf.WriteString("---\n\n")
	buf.WriteString(page.Body)

	if err := writeFileAtomic(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("ページの保存に失敗: %w", err)
	}

	cached := s.copyPage(page)
	if cached.CreatedAt.IsZero() {
		cached.CreatedAt = time.Now()
	}
	if cached.UpdatedAt.IsZero() {
		cached.UpdatedAt = time.Now()
	}
	s.cache[page.ID] = cached

	s.rebuildBacklinksForPage(page.ID, page.Body)

	return nil
}

// Delete は指定されたIDのページファイルを削除し、キャッシュからも除去する。
// 削除直前にファイル内容を履歴に退避する（管理者による復旧用）。
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dataDir, id+".md")

	if oldBytes, rerr := os.ReadFile(filePath); rerr == nil && len(oldBytes) > 0 && s.history != nil {
		if serr := s.history.Snapshot(id, oldBytes); serr != nil {
			_ = serr
		}
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("ページの削除に失敗: %w", err)
	}

	delete(s.cache, id)

	s.rebuildBacklinksForPage(id, "")

	return nil
}

// FilePath は指定されたIDのMarkdownファイルのフルパスを返す。
func (s *Store) FilePath(id string) string {
	return filepath.Join(s.dataDir, id+".md")
}

// DataDir はデータ保存ディレクトリのパスを返す。
func (s *Store) DataDir() string {
	return s.dataDir
}

// copyPage はPageのコピーを作成する。
// キャッシュから返すページが外部で変更されてもキャッシュが汚染されないようにコピーを返す。
func (s *Store) copyPage(p *Page) *Page {
	cp := *p
	if p.Tags != nil {
		cp.Tags = make([]string, len(p.Tags))
		copy(cp.Tags, p.Tags)
	}
	return &cp
}

// loadPageFromDisk は指定されたIDのMarkdownファイルをディスクから読み込み、
// Page構造体に変換する内部メソッド。起動時のキャッシュ初期化でのみ使用される。
func (s *Store) loadPageFromDisk(id string) (*Page, error) {
	filePath := filepath.Join(s.dataDir, id+".md")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ファイルの読み込みに失敗: %w", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("ファイル情報の取得に失敗: %w", err)
	}

	fm, body, perr := parsePageBytes(data)
	if perr != nil {
		return nil, perr
	}

	title := fm.Title
	if title == "" {
		title = id
	}

	createdAt := info.ModTime()
	updatedAt := info.ModTime()
	if fm.Created != nil {
		createdAt = *fm.Created
	}
	if fm.Update != nil {
		updatedAt = *fm.Update
	}

	return &Page{
		ID:        id,
		Title:     title,
		Body:      body,
		Tags:      fm.Tags,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// parsePageBytes はフロントマター付きMarkdownのバイト列をパースし、
// フロントマター（FrontMatter）と本文（フロントマター除去後）を返す。
// 通常のページ読み込みと履歴ファイル読み込みの両方から使われる。
func parsePageBytes(data []byte) (FrontMatter, string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(&frontmatter.Extender{}),
	)
	ctx := parser.NewContext()
	md.Parser().Parse(text.NewReader(data), parser.WithContext(ctx))

	var fm FrontMatter
	d := frontmatter.Get(ctx)
	if d != nil {
		if err := d.Decode(&fm); err != nil {
			return fm, "", fmt.Errorf("フロントマターの解析に失敗: %w", err)
		}
	}

	body := string(data)
	if strings.HasPrefix(body, "---") {
		idx := strings.Index(body[3:], "---")
		if idx >= 0 {
			body = strings.TrimSpace(body[3+idx+3:])
		}
	}

	return fm, body, nil
}

// extractWikiLinkTargets は本文中の全 WikiLink からリンク先ページIDを抽出する。
// [[ID]] または [[ID|表示テキスト]] の形式を検出し、ID 部分を重複なしで返す。
func extractWikiLinkTargets(body string) []string {
	matches := wikiLinkPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var targets []string
	for _, m := range matches {
		id := strings.TrimSpace(m[1])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			targets = append(targets, id)
		}
	}
	return targets
}

// buildBacklinksIndex は全ページの本文を走査してバックリンクインデックスを構築する。
// 起動時に一度だけ呼ばれる。
func (s *Store) buildBacklinksIndex() {
	for sourceID, p := range s.cache {
		targets := extractWikiLinkTargets(p.Body)
		for _, targetID := range targets {
			if targetID == sourceID {
				continue
			}
			s.backlinks[targetID] = append(s.backlinks[targetID], sourceID)
		}
	}
}

// rebuildBacklinksForPage は指定ページのバックリンクを差分更新する。
// このページがリンク元として登録されている全エントリを削除したあと、
// 新しい本文から WikiLink ターゲットを抽出して追加する。
// body が空文字の場合は削除のみ（ページ削除時に使う）。
// 呼び出し元で書き込みロック（s.mu.Lock）を取得済みであること。
func (s *Store) rebuildBacklinksForPage(sourceID string, body string) {
	for targetID, sources := range s.backlinks {
		filtered := make([]string, 0, len(sources))
		for _, sid := range sources {
			if sid != sourceID {
				filtered = append(filtered, sid)
			}
		}
		if len(filtered) > 0 {
			s.backlinks[targetID] = filtered
		} else {
			delete(s.backlinks, targetID)
		}
	}

	if body == "" {
		return
	}
	targets := extractWikiLinkTargets(body)
	for _, targetID := range targets {
		if targetID == sourceID {
			continue
		}
		s.backlinks[targetID] = append(s.backlinks[targetID], sourceID)
	}
}

// TagCounts は全ページのタグを集計し、タグ名→ページ数のマップを返す。
func (s *Store) TagCounts() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	for _, p := range s.cache {
		for _, tag := range p.Tags {
			counts[tag]++
		}
	}
	return counts
}

// ListAllTags は全ページで使われているタグをアルファベット順で返す。
func (s *Store) ListAllTags() []string {
	counts := s.TagCounts()
	tags := make([]string, 0, len(counts))
	for tag := range counts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// ListByTag は指定タグを持つページを更新日時の降順で返す。
func (s *Store) ListByTag(tag string) []*Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Page
	for _, p := range s.cache {
		for _, t := range p.Tags {
			if t == tag {
				result = append(result, s.copyPage(p))
				break
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result
}

// GraphNode はグラフ描画用のノード表現。クライアント（cytoscape.js）に渡される。
type GraphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// GraphEdge はグラフ描画用のエッジ表現。WikiLink の有向リンクに対応する。
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// BuildGraph は全ページをノード、WikiLink をエッジとしたグラフを返す。
// - リンクのない孤立ページもノードとして含める
// - 存在しないターゲットへのリンク（壊れたリンク）はエッジに含めない
// - 自己ループ（source==target）はエッジに含めない
// - 重複するエッジは1本にまとめる
func (s *Store) BuildGraph() ([]GraphNode, []GraphEdge) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]GraphNode, 0, len(s.cache))
	for _, p := range s.cache {
		nodes = append(nodes, GraphNode{ID: p.ID, Title: p.Title})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	type edgeKey struct{ src, tgt string }
	seen := make(map[edgeKey]struct{})
	edges := make([]GraphEdge, 0)

	for sourceID, p := range s.cache {
		targets := extractWikiLinkTargets(p.Body)
		for _, targetID := range targets {
			if targetID == sourceID {
				continue
			}
			if _, exists := s.cache[targetID]; !exists {
				continue
			}
			key := edgeKey{src: sourceID, tgt: targetID}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			edges = append(edges, GraphEdge{Source: sourceID, Target: targetID})
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})

	return nodes, edges
}

// writeFileAtomic はファイルをアトミックに書き込む。
// 一時ファイルに書き込み後、リネームで置換することで
// クラッシュ時にファイルが中途半端な状態になることを防ぐ。
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ivvvy-tmp-*")
	if err != nil {
		return fmt.Errorf("一時ファイルの作成に失敗: %w", err)
	}
	tmpPath := tmp.Name()

	// 失敗時のクリーンアップ
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("一時ファイルへの書き込みに失敗: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("一時ファイルの同期に失敗: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("一時ファイルのクローズに失敗: %w", err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("パーミッションの設定に失敗: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("ファイルの置換に失敗: %w", err)
	}

	tmpPath = "" // クリーンアップ不要
	return nil
}

// GetBacklinks は指定ページへのバックリンク元ページ一覧を返す。
// バックリンクインデックスからO(1)でリンク元IDを取得し、キャッシュからページ情報を返す。
// 結果はタイトルの昇順でソートされる。
func (s *Store) GetBacklinks(targetID string) ([]*Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sourceIDs, ok := s.backlinks[targetID]
	if !ok || len(sourceIDs) == 0 {
		return nil, nil
	}

	var result []*Page
	for _, sid := range sourceIDs {
		p, exists := s.cache[sid]
		if exists {
			result = append(result, s.copyPage(p))
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Title < result[j].Title
	})

	return result, nil
}
