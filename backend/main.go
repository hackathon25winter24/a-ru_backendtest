package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql" // MariaDB用
	"gorm.io/gorm"

	// TODO: go.mod のモジュール名に合わせて書き換えてください
	"my-app/pb"
)

// --- モデル定義 ---

type User struct {
	// MariaDBには専用のUUID型がないため、VARCHAR(36)として保存します
	ID    uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	Hash  string    `json:"hash"`
	Story int       `json:"story"`
	Rate  int       `json:"rate"`
}

// データを保存する直前にUUIDを自動生成するフック
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

// --- gRPC サーバー実装 ---

type server struct {
	pb.UnimplementedUserServiceServer
	db *gorm.DB
}

func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	user := User{
		Hash:  req.Hash,
		Story: int(req.Story),
		Rate:  int(req.Rate),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &pb.UserResponse{
		Id:    user.ID.String(),
		Hash:  user.Hash,
		Story: int32(user.Story),
		Rate:  int32(user.Rate),
	}, nil
}

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, err
	}

	var user User
	if err := s.db.First(&user, "id = ?", uid).Error; err != nil {
		return nil, err
	}

	return &pb.UserResponse{
		Id:    user.ID.String(),
		Hash:  user.Hash,
		Story: int32(user.Story),
		Rate:  int32(user.Rate),
	}, nil
}

// --- メイン処理 ---

func main() {
	// .envがあれば読み込む（ローカル用）、なければシステム環境変数を参照（NeoShowcase用）
	_ = godotenv.Load()

	// 1. 環境変数の取得
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT") // MariaDBなら通常 3306

	// 2. MariaDB (MySQL互換) 用の DSN 構築
	// user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// 3. DB接続
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to MariaDB: %v", err)
	}

	// テーブル自動作成
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database connection and migration successful.")

	// 4. gRPC サーバーの起動準備
	appPort := os.Getenv("PORT")
	if appPort == "" {
		appPort = "8080"
	}

	lis, err := net.Listen("tcp", ":"+appPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", appPort, err)
	}

	// 5. サーバー登録
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{db: db})
	reflection.Register(s) // grpcurl等での動作確認用

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}