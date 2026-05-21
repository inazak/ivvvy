package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/inazak/ivvvy/internal/markdown"
	"github.com/inazak/ivvvy/internal/page"
	"github.com/inazak/ivvvy/internal/search"
)

// Server はHTTPサーバー。ページの閲覧・編集・検索機能を提供する。
type Server struct {
	store    *page.Store
	renderer *markdown.Renderer
	indexer  *search.Indexer

	templateFS fs.FS

	// layout.html + 各ページテンプレートの組み合わせを個別にキャッシュする。
	// "content" ブロック名の競合を防ぐため、組み合わせごとにパースする。
	templateCache map[string]*template.Template
	cacheMu       sync.RWMutex

	staticFS fs.FS

	// 空の場合はIP制限を行わない。
	allowNets []*net.IPNet

	// 両方とも空の場合はBasic認証を行わない。
	basicAuthUser string
	basicAuthPass string
}

// Config はサーバー起動に必要な設定。
type Config struct {
	DataDir    string
	Port       int
	TemplateFS fs.FS
	StaticFS   fs.FS

	// 例: []string{"192.168.1.0/24", "127.0.0.1/32"}
	// 空（nil または長さ0）の場合はIP制限を行わない。
	// 単一IP指定でも CIDR 表記が必要（例: "10.0.0.5/32"）。
	AllowIPs []string

	// "user:pass" 形式。空文字列の場合はBasic認証を行わない。
	BasicAuth string
}

// New はサーバーインスタンスを生成する。
// 各コンポーネント（ストア、レンダラ、インデクサー）を初期化する。
func New(cfg Config) (*Server, error) {
	store, err := page.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("ページストアの初期化に失敗: %w", err)
	}

	indexer, err := search.NewIndexer()
	if err != nil {
		return nil, fmt.Errorf("検索インデクサーの初期化に失敗: %w", err)
	}

	// CIDR 表記ミスは起動時に検出して即エラーにする（運用中の意図せぬ全許可を防ぐ）。
	allowNets := make([]*net.IPNet, 0, len(cfg.AllowIPs))
	for _, s := range cfg.AllowIPs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, nw, perr := net.ParseCIDR(s)
		if perr != nil {
			return nil, fmt.Errorf("IP制限のCIDR表記が不正です（%q）: %w", s, perr)
		}
		allowNets = append(allowNets, nw)
	}

	var basicUser, basicPass string
	if cfg.BasicAuth != "" {
		idx := strings.IndexByte(cfg.BasicAuth, ':')
		if idx <= 0 || idx == len(cfg.BasicAuth)-1 {
			return nil, fmt.Errorf("Basic認証は \"user:pass\" 形式で指定してください")
		}
		basicUser = cfg.BasicAuth[:idx]
		basicPass = cfg.BasicAuth[idx+1:]
	}

	return &Server{
		store:         store,
		renderer:      markdown.NewRenderer(),
		indexer:       indexer,
		templateFS:    cfg.TemplateFS,
		templateCache: make(map[string]*template.Template),
		staticFS:      cfg.StaticFS,
		allowNets:     allowNets,
		basicAuthUser: basicUser,
		basicAuthPass: basicPass,
	}, nil
}

// templateFuncs はテンプレート内で使えるカスタム関数を返す。
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}
}

// Run はHTTPサーバーを起動する。ブロッキング呼び出し。
func (s *Server) Run(port int) error {
	r := s.routes()
	addr := fmt.Sprintf(":%d", port)
	log.Printf("サーバーを起動します: http://localhost%s", addr)
	return http.ListenAndServe(addr, r)
}

// routes はURLパスとハンドラの対応を定義する。
func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	if len(s.allowNets) > 0 {
		r.Use(s.ipAllowMiddleware)
	}

	if s.basicAuthUser != "" {
		r.Use(middleware.BasicAuth("ivvvy", map[string]string{
			s.basicAuthUser: s.basicAuthPass,
		}))
	}

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(s.staticFS))))

	r.Get("/", s.handleIndex)
	r.Get("/pages", s.handlePageList)
	r.Get("/page/{id}", s.handlePageView)
	r.Get("/page/{id}/edit", s.handlePageEdit)
	r.Get("/page/{id}/history", s.handlePageHistory)
	r.Get("/page/{id}/history/{filename}", s.handlePageVersionView)
	r.Get("/new", s.handlePageNew)
	r.Get("/tags", s.handleTagList)
	r.Get("/tags/{tag}", s.handleTagView)
	r.Get("/graph", s.handleGraphView)

	r.Get("/search/index.json", s.handleSearchIndex)

	r.Post("/api/page", s.handleAPIPageCreate)
	r.Put("/api/page/{id}", s.handleAPIPageUpdate)
	r.Delete("/api/page/{id}", s.handleAPIPageDelete)
	r.Get("/api/page/{id}/raw", s.handleAPIPageRaw)
	r.Post("/api/preview", s.handleAPIPreview)
	r.Get("/api/graph.json", s.handleAPIGraph)
	r.Get("/api/tags", s.handleAPITags)

	return r
}

// ipAllowMiddleware はリモートIPが allowNets のいずれかに含まれる場合のみ
// 次のハンドラへ進む。含まれなければ 403 を返す。
// RemoteAddr は "IP:port" 形式なので net.SplitHostPort で分解する。
// プロキシ越しの構成は想定外のため X-Forwarded-For は見ない。
func (s *Server) ipAllowMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		for _, nw := range s.allowNets {
			if nw.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}
