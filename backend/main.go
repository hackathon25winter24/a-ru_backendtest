package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"github.com/google/uuid"

	// 自動生成されたパッケージをインポート
	// "my-app" の部分は go.mod のモジュール名に合わせてください
	"my-app/pb"
)

// (前回の回答で定義したGORMモデル)
type User struct {
	ID    uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Hash  string
	Story int
	Rate  int
}

type server struct {
	pb.UnimplementedUserServiceServer
	db *gorm.DB
}

// CreateUser の実装
func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	// 1. リクエストからGORMモデルを作成
	// IDはDB側で自動生成させるのでここでは指定しない
	user := User{
		Hash:  req.Hash,
		Story: int(req.Story), // int32 -> int へのキャスト
		Rate:  int(req.Rate),
	}

	// 2. DBに保存
	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	// 3. レスポンスを返す (UUIDを文字列に変換)
	return &pb.UserResponse{
		Id:    user.ID.String(), // uuid.UUID -> string
		Hash:  user.Hash,
		Story: int32(user.Story),
		Rate:  int32(user.Rate),
	}, nil
}

// GetUser の実装
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	// 1. 文字列のIDを UUID型にパース（検証）
	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, err // 不正なUUID形式の場合
	}

	var user User
	// 2. 検索実行
	if err := s.db.First(&user, "id = ?", uid).Error; err != nil {
		return nil, err
	}

	// 3. レスポンス
	return &pb.UserResponse{
		Id:    user.ID.String(),
		Hash:  user.Hash,
		Story: int32(user.Story),
		Rate:  int32(user.Rate),
	}, nil
}

func main() {
	// 1. 設定読み込み & DB接続 (前回と同じ)
	godotenv.Load()
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	// マイグレーション（テーブル作成）
    db.AutoMigrate(&User{})

    // IDを指定せずに作成
    user := User{
        Hash:  "****",
        Story: 27,
        Rate:  2527,
    }
    
    // INSERT実行
    db.Create(&user)

    // 作成後は user.ID に自動生成されたUUIDが入っています
    fmt.Println("Generated ID:", user.ID.String())

    result := db.Create(&user)
    if result.Error != nil {
        // エラー処理
        panic(result.Error)
    }

	// 2. リスナーの作成 (TCPポートを開く)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 3. gRPCサーバーの作成
	s := grpc.NewServer()
	
	// 実装したサービスを登録 (DBを渡す)
	pb.RegisterUserServiceServer(s, &server{db: db})

	// サーバーリフレクションの設定 (grpcurlなどで動作確認しやすくするため)
	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}