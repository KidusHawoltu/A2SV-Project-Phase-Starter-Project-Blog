package main

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	"A2SV_Starter_Project_Blog/Delivery/routers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"A2SV_Starter_Project_Blog/config"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// --- Load Configuration ---
	cfg := config.Load()

	// --- Log important warnings based on the loaded config ---
	if cfg.GeminiAPIKey == "" {
		log.Println("WARN: GEMINI_API_KEY is not set. AI features will fail.")
	}
	if cfg.CloudinaryCloudName == "" || cfg.CloudinaryAPIKey == "" || cfg.CloudinaryAPISecret == "" {
		log.Println("WARN: Cloudinary credentials are not set. Image uploading will fail")
	}
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		log.Println("WARN: Google OAuth credentials are not set. Sign in with Google will fail.")
	}

	// --- MongoDB Setup ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(cfg.DBName)
	log.Println("MongoDB connected.")

	// --- Infrastructure Services ---
	// Pass values from the cfg struct to the service constructors.
	passwordService := infrastructure.NewPasswordService()
	jwtService := infrastructure.NewJWTService(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	emailService := infrastructure.NewSMTPEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	aiService, err := infrastructure.NewGeminiAIService(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Printf("WARN: Failed to initialize AI service: %v. AI features will be unavailable.", err)
	}
	googleOAuth2Service, err := infrastructure.NewGoogleOAuthService(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURI)
	if err != nil {
		log.Println("WARN: Google OAuth credentials are not set. Sign in with Google will fail.", err)
	}
	imageUploadService, err := infrastructure.NewCloudinaryService(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
	if err != nil {
		log.Printf("WARN: Cloudinary service failed to initialize. Image uploads will be unavailable. Error: %v", err)
	}
	redisService, err := infrastructure.NewRedisService(context.Background(), cfg.RedisUrl, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("FATAL: Redis connection failed.Error: %v", err)
	}
	defer redisService.Close()
	rateLimiter := infrastructure.NewRateLimiter(redisService)
	cacheService := infrastructure.NewRedisCacheService(redisService)

	// --- Repositories & Caching Decorators ---
	mongoUserRepo := repositories.NewMongoUserRepository(db, "users")
	userRepo := repositories.NewCachingUserRepository(mongoUserRepo, cacheService)

	mongoTokenRepo := repositories.NewMongoTokenRepository(db, "tokens")
	tokenRepo := repositories.NewCachingTokenRepository(mongoTokenRepo, cacheService)

	mongoBlogRepo := repositories.NewBlogRepository(db.Collection("blogs"))
	blogRepo := repositories.NewCachingBlogRepository(mongoBlogRepo, cacheService)

	mongoInteractionRepo := repositories.NewInteractionRepository(db.Collection("interactions"))
	interactionRepo := repositories.NewCachingInteractionRepository(mongoInteractionRepo, cacheService)

	mongoCommentRepo := repositories.NewCommentRepository(db.Collection("blog_comments"))
	commentRepo := repositories.NewCachingCommentRepository(mongoCommentRepo, cacheService)

	// --- Database Index Initialization ---
	log.Println("Initializing database indexes...")
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer indexCancel()

	handleIndexError := func(repoName string, err error) {
		if err != nil {
			if cfg.AppEnv == "production" {
				log.Fatalf("FATAL: failed to create %s indexes: %v", repoName, err)
			}
			log.Printf("WARN: failed to create %s indexes (The application may be slow): %v", repoName, err)
		}
	}
	handleIndexError("user", mongoUserRepo.CreateUserIndexes(indexCtx))
	handleIndexError("token", mongoTokenRepo.CreateTokenIndexes(indexCtx))
	handleIndexError("blog", mongoBlogRepo.CreateBlogIndexes(indexCtx))
	handleIndexError("interaction", mongoInteractionRepo.CreateInteractionIndexes(indexCtx))
	handleIndexError("comment", mongoCommentRepo.CreateCommentIndexes(indexCtx))
	log.Println("Database index initialization complete.")

	// --- Usecases ---
	userUsecase := usecases.NewUserUsecase(userRepo, passwordService, jwtService, tokenRepo, emailService, imageUploadService, cfg.UsecaseTimeout)
	blogUsecase := usecases.NewBlogUsecase(blogRepo, userRepo, interactionRepo, cfg.UsecaseTimeout)
	aiUsecase := usecases.NewAIUsecase(aiService, 10*cfg.UsecaseTimeout)
	commentUsecase := usecases.NewCommentUsecase(blogRepo, commentRepo, cfg.UsecaseTimeout)
	oauthUsecase := usecases.NewOAuthUsecase(userRepo, tokenRepo, jwtService, googleOAuth2Service, cfg.UsecaseTimeout)

	// --- Controllers & Router ---
	userController := controllers.NewUserController(userUsecase)
	blogController := controllers.NewBlogController(blogUsecase)
	aiController := controllers.NewAIController(aiUsecase)
	commentController := controllers.NewCommentController(commentUsecase)
	oauthController := controllers.NewOAuthController(oauthUsecase)

	router := routers.SetupRouter(userController, blogController, aiController, commentController, oauthController, jwtService, rateLimiter)

	log.Printf("Server starting on port %s...", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
