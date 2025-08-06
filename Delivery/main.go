package main

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	"A2SV_Starter_Project_Blog/Delivery/routers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load .env file from current or parent directory
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, proceeding with environment defaults...")
		}
	}

	// --- Environment Config ---
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("DB_NAME", "g6-blog-db")
	serverPort := getEnv("PORT", "8080")

	// JWT settings
	jwtSecret := getEnv("JWT_SECRET", "a-very-secret-key-that-should-be-long-and-random")
	jwtIssuer := "g6-blog-api"
	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MIN", "15"))
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_HR", "72"))
	jwtAccessTTL := time.Duration(accessTTL) * time.Minute
	jwtRefreshTTL := time.Duration(refreshTTL) * time.Hour

	// Usecase timeout
	usecaseTimeout := 5 * time.Second

	// Gemini AI settings
	geminiApiKey := getEnv("GEMINI_API_KEY", "")
	geminiModel := getEnv("GEMINI_MODEL", "gemini-2.5-pro")
	if geminiApiKey == "" {
		log.Println("WARN: GEMINI_API_KEY is not set. AI features will fail.")
	}

	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-2.5-pro"
	}
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		log.Println("WARN: GEMINI_API_KEY is not set. AI features will fail.")
	}
	usecaseTimeout := 5 * time.Second

	// SMTP email settings
	smtpHost := getEnv("SMTP_HOST", "smtp.mailtrap.io")
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "2525"))
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASSWORD", "")
	smtpFrom := getEnv("SMTP_FROM_EMAIL", "no-reply@example.com")

	// --- MongoDB Setup ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)

	log.Println("MongoDB connected.")

	// --- Infrastructure Services ---
	passwordService := infrastructure.NewPasswordService()
	jwtService := infrastructure.NewJWTService(jwtSecret, jwtIssuer, jwtAccessTTL, jwtRefreshTTL)
	emailService := infrastructure.NewSMTPEmailService(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom)
	aiService, err := infrastructure.NewGeminiAIService(geminiApiKey, geminiModel)
	if err != nil {
		log.Printf("WARN: Failed to initialize AI service: %v. AI features will be unavailable.", err)
	}

	// --- Repositories ---
	userRepo := repositories.NewMongoUserRepository(db, "users")
	tokenRepo := repositories.NewMongoTokenRepository(db, "tokens")
	blogRepo := repositories.NewBlogRepository(db.Collection("blogs"))
	interactionRepo := repositories.NewInteractionRepository(db.Collection("interactions"))

	// --- Usecases ---
	userUsecase := usecases.NewUserUsecase(userRepo, passwordService, jwtService, tokenRepo, emailService, usecaseTimeout)
	blogUsecase := usecases.NewBlogUsecase(blogRepo, userRepo, interactionRepo, usecaseTimeout)
	aiUsecase := usecases.NewAIUsecase(aiService)

	// --- Controllers ---
	userController := controllers.NewUserController(userUsecase)
	blogController := controllers.NewBlogController(blogUsecase)
	aiController := controllers.NewAIController(aiUsecase)


	// --- Router ---
	router := routers.SetupRouter(userController, blogController, aiController, jwtService)

	log.Printf("Server starting on port %s...", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// getEnv returns the env var value or a fallback if not set
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
