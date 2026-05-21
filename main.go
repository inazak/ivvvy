package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/inazak/ivvvy/internal/generate"
	"github.com/inazak/ivvvy/internal/server"
	"github.com/inazak/ivvvy/web/static"
	"github.com/inazak/ivvvy/web/template"
)

func main() {
	port := flag.Int("port", 3000, "HTTPサーバーのポート番号")
	dataDir := flag.String("data", "", "データ保存ディレクトリのパス（デフォルト: ./data）")
	allowIP := flag.String("allow-ip", "", "アクセス許可IPのCIDR（カンマ区切り、例: 192.168.1.0/24,127.0.0.1/32）。空なら制限なし")
	basicAuth := flag.String("basic-auth", "", "Basic認証 \"user:pass\" 形式。空なら認証なし")
	generateDir := flag.String("generate", "", "静的HTMLサイトを生成する出力ディレクトリ（指定するとサーバーは起動しない）")
	flag.Parse()

	// データディレクトリが未指定の場合は実行ファイルと同じ場所の data ディレクトリを使う
	if *dataDir == "" {
		exe, err := os.Executable()
		if err != nil {
			*dataDir = "data"
		} else {
			*dataDir = filepath.Join(filepath.Dir(exe), "data")
		}
	}

	if *generateDir != "" {
		gen, err := generate.NewGenerator(generate.GeneratorConfig{
			DataDir:    *dataDir,
			OutputDir:  *generateDir,
			TemplateFS: template.TemplateFS,
			StaticFS:   static.StaticFS,
		})
		if err != nil {
			log.Fatalf("静的サイトジェネレータの初期化に失敗しました: %v", err)
		}
		if err := gen.Run(); err != nil {
			log.Fatalf("静的サイトの生成に失敗しました: %v", err)
		}
		fmt.Printf("静的サイトを生成しました: %s\n", *generateDir)
		return
	}

	// -allow-ip はカンマ区切り。空要素は New 側で除去する。
	var allowIPs []string
	if *allowIP != "" {
		allowIPs = strings.Split(*allowIP, ",")
	}

	cfg := server.Config{
		DataDir:    *dataDir,
		Port:       *port,
		TemplateFS: template.TemplateFS,
		StaticFS:   static.StaticFS,
		AllowIPs:   allowIPs,
		BasicAuth:  *basicAuth,
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("サーバーの初期化に失敗しました: %v", err)
	}

	fmt.Printf("ivvvy を起動します\n")
	fmt.Printf("  データディレクトリ: %s\n", *dataDir)
	fmt.Printf("  URL: http://localhost:%d\n", *port)
	if len(allowIPs) > 0 {
		fmt.Printf("  IP制限: %s\n", *allowIP)
	}
	if *basicAuth != "" {
		fmt.Printf("  Basic認証: 有効\n")
	}

	if err := srv.Run(*port); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}
