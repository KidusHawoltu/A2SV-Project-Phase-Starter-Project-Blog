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
	appEnv := getEnv("APP_ENV", "development")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("DB_NAME", "g6-blog-db")
	serverPort := getEnv("PORT", "8080")

	// -- Redis
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "") // Default to no password
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

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

	// Cloudinary
	cloudinaryCloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	cloudinaryApiKey := os.Getenv("CLOUDINARY_API_KEY")
	cloudinaryApiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	if cloudinaryCloudName == "" || cloudinaryApiKey == "" || cloudinaryApiSecret == "" {
		log.Panicln("WARN: Cloudinary credentials are not set. Image uploading will fail")
	}

	// Google OAuth2 Credentials
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if googleClientID == "" || googleClientSecret == "" {
		log.Println("WARN: Google OAuth credentials are not set. Sign in with Google will fail.")
	}

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
	googleOAuth2Service, err := infrastructure.NewGoogleOAuthService(googleClientID, googleClientSecret, googleRedirectURI)
	if err != nil {
		log.Println("WARN: Google OAuth credentials are not set. Sign in with Google will fail.", err)
	}
	imageUploadService, err := infrastructure.NewCloudinaryService(cloudinaryCloudName, cloudinaryApiKey, cloudinaryApiSecret)
	if err != nil {
		log.Printf("WARN: Cloudinary service failed to initialize. Image uploads will be unavailable. Error: %v", err)
	}
	redisService, err := infrastructure.NewRedisService(context.Background(), redisAddr, redisPassword, redisDB)
	if err != nil {
		log.Fatalf("FATAL: Redis connection failed.Error: %v", err)
	}
	defer redisService.Close()
	rateLimiter := infrastructure.NewRateLimiter(redisService)

	// --- Repositories ---
	userRepo := repositories.NewMongoUserRepository(db, "users")
	tokenRepo := repositories.NewMongoTokenRepository(db, "tokens")
	blogRepo := repositories.NewBlogRepository(db.Collection("blogs"))
	interactionRepo := repositories.NewInteractionRepository(db.Collection("interactions"))
	commentRepo := repositories.NewCommentRepository(db.Collection("blog_comments"))

	// --- Database Index Initialization ---
	log.Println("Initializing database indexes...")
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer indexCancel()

	handleIndexError := func(repoName string, err error) {
		if err != nil {
			// In production, a failure to create an index is a fatal error.
			if appEnv == "production" {
				log.Fatalf("FATAL: failed to create %s indexes: %v", repoName, err)
			}
			// In development or other environments, just log a warning.
			log.Printf("WARN: failed to create %s indexes (The application may be slow): %v", repoName, err)
		}
	}

	handleIndexError("user", userRepo.CreateUserIndexes(indexCtx))
	handleIndexError("token", tokenRepo.CreateTokenIndexes(indexCtx))
	handleIndexError("blog", blogRepo.CreateBlogIndexes(indexCtx))
	handleIndexError("interaction", interactionRepo.CreateInteractionIndexes(indexCtx))
	handleIndexError("comment", commentRepo.CreateCommentIndexes(indexCtx))

	log.Println("Database index initialization complete.")

	// --- Usecases ---
	userUsecase := usecases.NewUserUsecase(userRepo, passwordService, jwtService, tokenRepo, emailService, imageUploadService, usecaseTimeout)
	blogUsecase := usecases.NewBlogUsecase(blogRepo, userRepo, interactionRepo, usecaseTimeout)
	aiUsecase := usecases.NewAIUsecase(aiService, 5*usecaseTimeout)
	commentUsecase := usecases.NewCommentUsecase(blogRepo, commentRepo, usecaseTimeout)
	oauthUsecase := usecases.NewOAuthUsecase(userRepo, tokenRepo, jwtService, googleOAuth2Service, usecaseTimeout)

	// --- Controllers ---
	userController := controllers.NewUserController(userUsecase)
	blogController := controllers.NewBlogController(blogUsecase)
	aiController := controllers.NewAIController(aiUsecase)
	commentController := controllers.NewCommentController(commentUsecase)
	oauthController := controllers.NewOAuthController(oauthUsecase)

	// --- Setup Router ---
	router := routers.SetupRouter(userController, blogController, aiController, commentController, oauthController, jwtService, rateLimiter)

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
