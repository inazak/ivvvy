// generate パッケージは ivvvy の静的サイトジェネレータ機能を提供する。
// CLIオプション -generate で指定された出力ディレクトリに、
// 全ページを静的HTMLファイルとして出力する。
// 生成されたサイトでは新規作成メニューや編集ボタンが非表示になる。
package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/inazak/ivvvy/internal/markdown"
	"github.com/inazak/ivvvy/internal/page"
	"github.com/inazak/ivvvy/internal/search"
	"github.com/inazak/ivvvy/internal/server"
)

// GeneratorConfig は静的サイト生成に必要な設定をまとめた構造体。
type GeneratorConfig struct {
	DataDir    string
	OutputDir  string
	TemplateFS fs.FS
	StaticFS   fs.FS
}

// Generator は全ページを静的HTMLファイルとして出力するジェネレータ。
type Generator struct {
	store      *page.Store
	renderer   *markdown.Renderer
	indexer    *search.Indexer
	templateFS fs.FS
	staticFS   fs.FS
	outputDir  string
}

// NewGenerator は静的サイトジェネレータのインスタンスを生成する。
func NewGenerator(cfg GeneratorConfig) (*Generator, error) {
	store, err := page.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ページストアの初期化に失敗: %w", err)
	}

	indexer, err := search.NewIndexer()
	if err != nil {
		return nil, fmt.Errorf("検索インデクサーの初期化に失敗: %w", err)
	}

	return &Generator{
		store:      store,
		renderer:   markdown.NewRenderer(),
		indexer:    indexer,
		templateFS: cfg.TemplateFS,
		staticFS:   cfg.StaticFS,
		outputDir:  cfg.OutputDir,
	}, nil
}

// Run は静的サイト生成のメインエントリーポイント。
func (g *Generator) Run() error {
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗: %w", err)
	}

	if err := g.copyStaticAssets(); err != nil {
		return fmt.Errorf("静的アセットのコピーに失敗: %w", err)
	}

	pages, err := g.store.List()
	if err != nil {
		return fmt.Errorf("ページ一覧の取得に失敗: %w", err)
	}

	for _, p := range pages {
		if err := g.generatePage(p, pages); err != nil {
			return fmt.Errorf("ページ %s の生成に失敗: %w", p.ID, err)
		}
	}

	if err := g.renderToFile("index.html", "index.html", server.TemplateData{
		Title:    "ivvvy",
		IsStatic: true,
	}); err != nil {
		return fmt.Errorf("トップページの生成に失敗: %w", err)
	}

	if err := g.renderToFile("list.html", filepath.Join("pages", "index.html"), server.TemplateData{
		Title:    "全ページ一覧",
		Pages:    pages,
		IsStatic: true,
	}); err != nil {
		return fmt.Errorf("ページ一覧の生成に失敗: %w", err)
	}

	if err := g.generateTagPages(); err != nil {
		return fmt.Errorf("タグページの生成に失敗: %w", err)
	}

	if err := g.generateGraphPage(); err != nil {
		return fmt.Errorf("グラフビューの生成に失敗: %w", err)
	}

	if err := g.generateSearchIndex(pages); err != nil {
		return fmt.Errorf("検索インデックスの生成に失敗: %w", err)
	}

	return nil
}

// generatePage は個別ページのHTMLを生成する。
func (g *Generator) generatePage(p *page.Page, allPages []*page.Page) error {
	html, err := g.renderer.Render([]byte(p.Body))
	if err != nil {
		return fmt.Errorf("Markdownの変換に失敗: %w", err)
	}

	backlinks, _ := g.store.GetBacklinks(p.ID)

	// page/{id}/index.html として出力する（ディレクトリベースURLでクリーンURLを維持）
	outPath := filepath.Join("page", p.ID, "index.html")
	return g.renderToFile("view.html", outPath, server.TemplateData{
		Title:         p.Title,
		Page:          p,
		Content:       string(html),
		BacklinkPages: backlinks,
		IsStatic:      true,
	})
}

// generateGraphPage はグラフビュー画面とグラフデータJSONを生成する。
// HTMLはクライアント側で /api/graph.json をfetchして cytoscape.js で描画する仕組み。
func (g *Generator) generateGraphPage() error {
	if err := g.renderToFile("graph.html", filepath.Join("graph", "index.html"), server.TemplateData{
		Title:    "グラフビュー",
		IsStatic: true,
	}); err != nil {
		return err
	}

	nodes, edges := g.store.BuildGraph()
	data, err := json.Marshal(map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
	if err != nil {
		return fmt.Errorf("グラフJSONのエンコードに失敗: %w", err)
	}

	outPath := filepath.Join(g.outputDir, "api", "graph.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

// generateSearchIndex は検索用のインデックスJSONファイルを生成する。
func (g *Generator) generateSearchIndex(pages []*page.Page) error {
	data, err := g.indexer.BuildIndex(pages)
	if err != nil {
		return fmt.Errorf("検索インデックスの生成に失敗: %w", err)
	}

	outPath := filepath.Join(g.outputDir, "search", "index.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

// generateTagPages はタグ一覧ページと個別タグページを生成する。
func (g *Generator) generateTagPages() error {
	counts := g.store.TagCounts()

	tagCounts := make([]server.TagCount, 0, len(counts))
	for name, count := range counts {
		tagCounts = append(tagCounts, server.TagCount{Name: name, Count: count})
	}
	sort.Slice(tagCounts, func(i, j int) bool {
		return tagCounts[i].Name < tagCounts[j].Name
	})

	if err := g.renderToFile("tags.html", filepath.Join("tags", "index.html"), server.TemplateData{
		Title:     "タグ一覧",
		TagCounts: tagCounts,
		IsStatic:  true,
	}); err != nil {
		return fmt.Errorf("タグ一覧ページの生成に失敗: %w", err)
	}

	for _, tc := range tagCounts {
		pages := g.store.ListByTag(tc.Name)
		if err := g.renderToFile("tag_pages.html", filepath.Join("tags", tc.Name, "index.html"), server.TemplateData{
			Title:    "タグ: " + tc.Name,
			Tag:      tc.Name,
			Pages:    pages,
			IsStatic: true,
		}); err != nil {
			return fmt.Errorf("タグ %s のページ生成に失敗: %w", tc.Name, err)
		}
	}

	return nil
}

// copyStaticAssets は埋め込みファイルシステムから静的アセットを出力ディレクトリにコピーする。
func (g *Generator) copyStaticAssets() error {
	return fs.WalkDir(g.staticFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(g.staticFS, path)
		if err != nil {
			return fmt.Errorf("静的ファイルの読み込みに失敗 (%s): %w", path, err)
		}

		outPath := filepath.Join(g.outputDir, "static", path)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(outPath, data, 0644)
	})
}

// renderToFile はテンプレートを実行してHTMLファイルとして出力する。
func (g *Generator) renderToFile(templateName string, outRelPath string, data server.TemplateData) error {
	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(g.templateFS, "layout.html", templateName)
	if err != nil {
		return fmt.Errorf("テンプレートのパースに失敗 (%s): %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		return fmt.Errorf("テンプレートの実行に失敗 (%s): %w", templateName, err)
	}

	outPath := filepath.Join(g.outputDir, outRelPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outPath, buf.Bytes(), 0644)
}
