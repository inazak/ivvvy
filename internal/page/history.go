package page

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HistoryStore はページの版管理（過去ファイルの保管・参照）を担当する。
//
// ファイル構造:
//
//	{baseDir}/{ID}/{UnixMilliTimestamp}.md
//
// ファイル名は UnixMilli タイムスタンプで、ファイル本体はフロントマター込みの
// 完全な .md ファイル（Store.Save が書き出す形式と同じ）。
type HistoryStore struct {
	baseDir string

	// Snapshot 時のタイムスタンプ衝突を回避するためのミューテックス。
	// 同一ミリ秒内に連続して呼ばれた場合にファイル名を 1ms ずつずらす処理を保護する。
	mu sync.Mutex
}

// VersionEntry はある1つの過去バージョンを表す構造体。
type VersionEntry struct {
	// FileName はディレクトリ内のファイル名（拡張子含む）。例: "1734567890123.md"
	FileName string

	// Timestamp はファイル名から解釈した時刻（UnixMilli から復元）。
	Timestamp time.Time
}

// NewHistoryStore は履歴ストアを初期化する。baseDir が存在しなければ作成する。
func NewHistoryStore(baseDir string) (*HistoryStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("履歴ディレクトリの作成に失敗: %w", err)
	}
	return &HistoryStore{baseDir: baseDir}, nil
}

// Snapshot は指定IDの「現在の」ファイル内容を1つの版として保存する。
// Store.Save の「上書き直前」のタイミングで呼ばれることを想定している。
//
// ファイル名は現在時刻（UnixMilli）をベースとする。同一ミリ秒内に
// 衝突した場合は 1ms ずつインクリメントして空いている名前を探す。
// fileBytes が空の場合は何もしない（新規作成時には版を残さない）。
func (h *HistoryStore) Snapshot(id string, fileBytes []byte) error {
	if id == "" {
		return fmt.Errorf("IDが空です")
	}
	if len(fileBytes) == 0 {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	dir := filepath.Join(h.baseDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("履歴サブディレクトリの作成に失敗: %w", err)
	}

	ts := time.Now().UnixMilli()
	for {
		filename := strconv.FormatInt(ts, 10) + ".md"
		full := filepath.Join(dir, filename)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			if werr := writeFileAtomic(full, fileBytes, 0644); werr != nil {
				return fmt.Errorf("履歴ファイルの書き込みに失敗: %w", werr)
			}
			return nil
		}
		ts++
	}
}

// ListVersions は指定IDの全バージョンをタイムスタンプ降順（新しい順）で返す。
// 該当ディレクトリがない場合は nil を返す（エラーにしない）。
func (h *HistoryStore) ListVersions(id string) ([]VersionEntry, error) {
	if id == "" {
		return nil, fmt.Errorf("IDが空です")
	}

	dir := filepath.Join(h.baseDir, id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("履歴ディレクトリの読み込みに失敗: %w", err)
	}

	versions := make([]VersionEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".md")
		ms, perr := strconv.ParseInt(base, 10, 64)
		if perr != nil {
			continue
		}
		versions = append(versions, VersionEntry{
			FileName:  e.Name(),
			Timestamp: time.UnixMilli(ms),
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Timestamp.After(versions[j].Timestamp)
	})

	return versions, nil
}

// LoadVersion は指定IDの指定ファイル名のバージョンを Page として読み込む。
// 過去のフロントマター+本文をパースして当時のタイトルを再現する。
//
// セキュリティ: filename はディレクトリトラバーサル回避のため「数字.md」形式のみ許可する。
func (h *HistoryStore) LoadVersion(id, filename string) (*Page, error) {
	if id == "" {
		return nil, fmt.Errorf("IDが空です")
	}
	if !strings.HasSuffix(filename, ".md") {
		return nil, fmt.Errorf("不正なファイル名です: %s", filename)
	}
	base := strings.TrimSuffix(filename, ".md")
	ms, perr := strconv.ParseInt(base, 10, 64)
	if perr != nil {
		return nil, fmt.Errorf("不正なファイル名です: %s", filename)
	}

	full := filepath.Join(h.baseDir, id, filename)
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("履歴ファイルの読み込みに失敗: %w", err)
	}

	fm, body, perr := parsePageBytes(data)
	if perr != nil {
		return nil, fmt.Errorf("履歴ファイルの解析に失敗: %w", perr)
	}

	title := fm.Title
	if title == "" {
		title = id
	}

	ts := time.UnixMilli(ms)

	return &Page{
		ID:        id,
		Title:     title,
		Body:      body,
		CreatedAt: ts,
		UpdatedAt: ts,
	}, nil
}
