package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/inazak/ivvvy/internal/page"
	webtmpl "github.com/inazak/ivvvy/web/template"
)

// newTestServer はテスト用のServerインスタンスを生成するヘルパー関数。
// server.New() は kagome 辞書のロードなど重い処理を伴うため、
// ハンドラ単体のテストでは最低限必要なフィールドのみを直接埋めて返す。
func newTestServer(t *testing.T) *Server {
	t.Helper()

	store, err := page.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("ページストアの初期化に失敗: %v", err)
	}

	return &Server{
		store:         store,
		templateFS:    webtmpl.TemplateFS,
		templateCache: make(map[string]*template.Template),
		cacheMu:       sync.RWMutex{},
	}
}

// TestHandleGraphView は、/graph 画面のHTMLが正しくレンダリングされ、
// cytoscape.js と graph.js のスクリプト参照を含むことを検証する。
func TestHandleGraphView(t *testing.T) {
	srv := newTestServer(t)

	r := chi.NewRouter()
	r.Get("/graph", srv.handleGraphView)

	req := httptest.NewRequest(http.MethodGet, "/graph", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコードが200ではない: got=%d body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, expect := range []string{"cytoscape.min.js", "graph.js", `id="cy"`} {
		if !strings.Contains(body, expect) {
			t.Errorf("レスポンス本文に %q が含まれていない", expect)
		}
	}
}

// TestHandleAPIGraph は、グラフAPI が
// 全ページをノードとして返し、WikiLink をエッジとして返すことを検証する。
func TestHandleAPIGraph(t *testing.T) {
	srv := newTestServer(t)

	srv.store.Save(&page.Page{ID: "X", Title: "ページX", Body: "[[Y]]"})
	srv.store.Save(&page.Page{ID: "Y", Title: "ページY", Body: "本文のみ"})

	r := chi.NewRouter()
	r.Get("/api/graph.json", srv.handleAPIGraph)

	req := httptest.NewRequest(http.MethodGet, "/api/graph.json", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ステータスコードが200ではない: got=%d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type が application/json ではない: got=%q", ct)
	}

	var got struct {
		Nodes []map[string]string `json:"nodes"`
		Edges []map[string]string `json:"edges"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("レスポンスのJSONパースに失敗: %v", err)
	}

	if len(got.Nodes) != 2 {
		t.Errorf("ノード数が不一致: got=%d, want=2", len(got.Nodes))
	}
	if len(got.Edges) != 1 {
		t.Errorf("エッジ数が不一致: got=%d, want=1", len(got.Edges))
	}
	if len(got.Edges) == 1 {
		e := got.Edges[0]
		if e["source"] != "X" || e["target"] != "Y" {
			t.Errorf("エッジが不一致: got=%v, want={source:X, target:Y}", e)
		}
	}
}
