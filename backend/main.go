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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// ↓ go.mod のモジュール名に合わせてください
	"my-app/pb"
)

// User モデル定義
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

// GetUser の実装
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

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found, relying on system environment variables")
	}

	// 1. DB接続情報の取得
	host := os.Getenv("DB_HOST")
	// 【修正1】変数名を dbUser に変更（構造体の user と被らないように）
	dbUser := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT") // ここは dbPort としておくのが無難
	sslmode := os.Getenv("DB_SSLMODE")

	if sslmode == "" {
		sslmode = "disable"
	}

	// DSN の構築 (変数を dbUser, dbPort に変更)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, dbUser, password, dbname, dbPort, sslmode)

	// 2. DB接続
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	
	// マイグレーション
	db.AutoMigrate(&User{})

	// --- 起動時のデータ作成テスト (不要なら削除可) ---
	// ここでの user は構造体のインスタンスなのでOK
	demoUser := User{
		Hash:  "****",
		Story: 27,
		Rate:  2527,
	}
	
	// INSERT実行
	result := db.Create(&demoUser)
	if result.Error != nil {
		panic(result.Error)
	}
	fmt.Println("Generated ID:", demoUser.ID.String())
	// ------------------------------------------

	// 3. リスナーの作成
	// 【修正2】変数名を appPort に変更（上の dbPort と被らないように）
	appPort := os.Getenv("PORT")
	if appPort == "" {
		appPort = "8080"
	}
	
	lis, err := net.Listen("tcp", ":"+appPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 4. gRPCサーバーの起動
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{db: db})
	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}