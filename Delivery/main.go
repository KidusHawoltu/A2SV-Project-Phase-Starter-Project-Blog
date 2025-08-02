package main

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	"A2SV_Starter_Project_Blog/Delivery/routers"
	"A2SV_Starter_Project_Blog/Infrastructure"
	"A2SV_Starter_Project_Blog/Repositories"
	"A2SV_Starter_Project_Blog/Usecases"
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, proceeding with environment defaults...")
		}
	}

	// Configurations
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "g6-blog-db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "a-very-secret-key-that-should-be-long-and-random"
	}

	jwtIssuer := "g6-blog-api"
	jwtTTL := 15 * time.Minute
	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	usecaseTimeout := 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// --- MongoDB Setup ---
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB (ping failed): %v", err)
	}

	defer client.Disconnect(context.Background())
	db := client.Database(dbName)

	log.Println("MongoDB connected.")

	// --- Infrastructure Setup ---
	passwordService := infrastructure.NewPasswordService()
	jwtService := infrastructure.NewJWTService(jwtSecret, jwtIssuer, jwtTTL)

	// --- Repositories ---
	userRepo := repositories.NewMongoUserRepository(db, "users")
	blogRepo := repositories.NewBlogRepository(db.Collection("blogs"))

	// --- Usecases ---
	userUsecase := usecases.NewUserUsecase(userRepo, passwordService, jwtService, usecaseTimeout)
	blogUsecase := usecases.NewBlogUsecase(blogRepo, userRepo, usecaseTimeout)

	// --- Controllers ---
	userController := controllers.NewUserController(userUsecase)
	blogController := controllers.NewBlogController(blogUsecase)

	// --- Setup Router ---
	router := routers.SetupRouter(userController, blogController, jwtService)

	log.Printf("Server starting on port %s...", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
