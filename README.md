# ivvvy

Markdown形式のフラットファイルをブラウザ上で閲覧・編集・検索できる、テキスト特化のドキュメント管理ツール。Go製シングルバイナリとして配布可能。

## 主な機能

### ページ管理
- YAMLフロントマター（title, tags）付きMarkdownファイルとして保存
- ページIDはミリ秒タイムスタンプの自動採番
- 作成・更新・削除・Markdownソース取得をREST APIで操作
- ページ保存時に過去バージョンを `history/{ID}/{timestamp}.md` へ自動退避

### ナビゲーション
- **グラフビュー**: ページ間のWikiLinkを有向グラフとして可視化（ivy=ツタの由来にちなんで緑基調）
- **バックリンク**: そのページへWikiLinkを張っているページを逆引き表示
- **タグ**: ページにタグを付与し、タグ一覧・タグ別ページ絞り込みが可能

### WikiLink
- `[[ページID]]` 記法でページ間リンクを記述
- `[[ID|表示テキスト]]` パイプ記法でリンクテキストを指定
- キーワード記法 `[[タイトル]]` を書いて保存すると、一致するページが存在する時点で `[[ID|タイトル]]` に自動書き換え

### 全文検索
- サーバー側でkagome（IPA辞書）により形態素解析してトークン化
- ブラウザがJSONインデックスを取得してAND方式のクライアントサイド検索を実行
- 検索結果にキーワード周辺のスニペット（前後40文字）をハイライト表示

### Markdown拡張
- GitHub Flavored Markdown（テーブル・タスクリスト・取り消し線）
- Obsidian/Quartz形式のCalloutブロック（`> [!note]` / `> [!warning]` など10タイプ）
- サーバーサイドプレビューAPI（閲覧時と同一のgoldmarkで変換）

### UI
- ライト／ダークモード切替（月/太陽アイコン）
- テキストサイズ3段階切替（large 19px / normal 16px / small 13px）
- キーボードショートカット（Gmail風の `g p` / `g r` 等、Vim風の `j` / `k` / `G`）
- 印刷用スタイル（ナビ非表示・背景透明化）
- レスポンシブ対応（768px以下でモバイルレイアウト）

### アクセス保護
- `-allow-ip` フラグによるIPアドレス制限（CIDR表記）
- `-basic-auth "user:pass"` フラグによるBasic認証
- 両方を同時に有効にするとAND条件で制限

### 静的サイト生成
`-generate <outdir>` フラグを指定するとサーバーを起動せずに全ページを静的HTMLとして出力。クライアントサイド検索もそのまま動作する。

## 起動方法

```bash
# デフォルト設定（ポート3000、実行ファイルと同じ場所の data ディレクトリ）
./ivvvy

# カスタム設定
./ivvvy -port 8080 -data /path/to/documents

# IP制限 + Basic認証
./ivvvy -allow-ip "192.168.0.0/16" -basic-auth "user:pass"

# 静的サイト生成
./ivvvy -generate ./out -data /path/to/documents
```

## ビルド

```bash
go build -o ivvvy .
go test ./internal/... -v
```

## アーキテクチャ概要

| 領域 | 技術 |
|------|------|
| 言語 | Go 1.26+ |
| HTTPサーバー | net/http + go-chi/chi v5 |
| Markdown変換 | yuin/goldmark |
| フロントマター | go.abhg.dev/goldmark/frontmatter |
| 形態素解析 | ikawaha/kagome v2 (IPA辞書) |
| グラフ描画 | cytoscape.js v3（クライアントサイド） |
| 静的アセット | Go embed |

全ページをオンメモリキャッシュ（`map[string]*Page`）に保持し、読み取り時のディスクI/Oを排除している。データベースは使用せず、バックアップ・マイグレーションが単純なファイルコピーで済む。
