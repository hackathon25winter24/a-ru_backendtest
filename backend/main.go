package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// サンプルモデル
type User struct {
	gorm.Model
	Name  string
	Email string
}

func main() {
	// 1. 環境変数から接続情報を取得（NeoShowcaseで設定される値）
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")
	sslmode := os.Getenv("DB_SSLMODE") // 通常は "disable"

	if sslmode == "" {
		sslmode = "disable"
	}

	// DSN (Data Source Name) の構築
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)

	// 2. DB接続
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 3. マイグレーション（テーブル自動作成）
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}
	
	fmt.Println("Database connection successful & migrated!")

	// 4. HTTPサーバーの起動（ヘルスチェック用）
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, NeoShowcase!")
	})

	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		portEnv = "8080"
	}

	log.Printf("Server starting on port %s", portEnv)
	if err := http.ListenAndServe(":"+portEnv, nil); err != nil {
		log.Fatal(err)
	}
}