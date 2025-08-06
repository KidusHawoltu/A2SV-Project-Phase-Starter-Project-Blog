package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"
	"A2SV_Starter_Project_Blog/Delivery/controllers" // Adjust import paths as needed
	"A2SV_Starter_Project_Blog/Delivery/routers"
	"A2SV_Starter_Project_Blog/Infrastructure"
	"A2SV_Starter_Project_Blog/Repositories"
	"A2SV_Starter_Project_Blog/Usecases"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load .env file (from remote) - this is a good pattern
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, proceeding with environment defaults...")
		}
	}

	// --- Configurations ---
	// Using the helper function for cleanliness
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("DB_NAME", "g6-blog-db")
	serverPort := getEnv("PORT", "8080")
	usecaseTimeout := 5 * time.Second

	// JWT Config (MERGED) - Using separate TTLs from your local logic
	jwtSecret := getEnv("JWT_SECRET", "default-secret-key-please-change")
	jwtIssuer := "g6-blog-api"
	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MIN", "15"))
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_HR", "72"))
	jwtAccessTTL := time.Duration(accessTTL) * time.Minute
	jwtRefreshTTL := time.Duration(refreshTTL) * time.Hour

	// AI Config (from remote)
	geminiApiKey := getEnv("GEMINI_API_KEY", "")
	geminiModel := getEnv("GEMINI_MODEL", "gemini-pro")
	if geminiApiKey == "" {
		log.Println("WARN: GEMINI_API_KEY is not set. AI features will fail.")
	}
    
    // SMTP Config (from your local logic, needed for Phase 3)
	smtpHost := getEnv("SMTP_HOST", "smtp.mailtrap.io")
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "2525"))
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASSWORD", "")
	smtpFrom := getEnv("SMTP_FROM_EMAIL", "no-reply@example.com")


	// --- Database Connection (from remote) ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Using a shorter, more reasonable timeout
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)
	log.Println("MongoDB connected.")

	// --- Dependency Injection (MERGED) ---

	// Infrastructure (All services from both versions)
	passwordService := infrastructure.NewPasswordService()
	// **MERGED**: Pass both TTLs to JWT Service
	jwtService := infrastructure.NewJWTService(jwtSecret, jwtIssuer, jwtAccessTTL, jwtRefreshTTL)
	// **ADDED from local**: Instantiate the SMTP service
	emailService := infrastructure.NewSMTPEmailService(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom)
	// **KEPT from remote**: AI Service
	aiService, err := infrastructure.NewGeminiAIService(geminiApiKey, geminiModel)
	if err != nil {
		log.Printf("WARN: Failed to initialize AI service: %v. AI features will be unavailable.", err)
	}

	// Repositories (All repos from both versions)
	userRepo := repositories.NewMongoUserRepository(db, "users")
	// **ADDED from local**: Token Repository is needed for Phase 3
	tokenRepo := repositories.NewMongoTokenRepository(db, "tokens") 
	blogRepo := repositories.NewBlogRepository(db, "blogs")
	interactionRepo := repositories.NewInteractionRepository(db, "interactions")

	// Usecases (All usecases from both versions)
	// **MERGED**: Pass all dependencies to the UserUsecase constructor for full Phase 3 functionality
	userUsecase := usecases.NewUserUsecase(userRepo, passwordService, jwtService, tokenRepo, emailService, usecaseTimeout)
	// **KEPT from remote**: Blog and AI usecases
	blogUsecase := usecases.NewBlogUsecase(blogRepo, userRepo, interactionRepo, usecaseTimeout)
	aiUsecase := usecases.NewAIUsecase(aiService)

	// Controllers (All controllers from both versions)
	userController := controllers.NewUserController(userUsecase)
	blogController := controllers.NewBlogController(blogUsecase)
	aiController := controllers.NewAIController(aiUsecase)

	// --- Router Setup (from remote, as it's more complete) ---
	router := routers.SetupRouter(userController, blogController, aiController, jwtService)

	log.Printf("Server starting on port %s...", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// getEnv is a helper function to read an environment variable or return a default value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}