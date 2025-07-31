package main

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	"A2SV_Starter_Project_Blog/Delivery/routers"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// --- Configuration ---
	
	mongoURI := "mongodb://localhost:27017"
	dbName := "g6-blog-db"
	userCollection := "users"
	jwtSecret := "a-very-secret-key-that-should-be-long-and-random"
	jwtIssuer := "g6-blog-api"
	jwtAccessTokenTTL := 15 * time.Minute // 15 minutes
	serverPort := ":8080"
	contextTimeout := 2 * time.Second

	// --- Database Connection ---
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(dbName)
	log.Println("Successfully connected to MongoDB.")
	
	// --- Dependency Injection ---

	// Infrastructure
	passwordService := infrastructure.NewPasswordService()
	jwtService := infrastructure.NewJWTService(jwtSecret, jwtIssuer, jwtAccessTokenTTL)

	// Repositories
	userRepository := repositories.NewMongoUserRepository(db, userCollection)

	// Usecases
	userUsecase := usecases.NewUserUsecase(userRepository, passwordService, jwtService, contextTimeout)

	// Controllers
	userController := controllers.NewUserController(userUsecase)

	// --- Router Setup ---
	router := routers.SetupRouter(userController, jwtService)
	
	log.Printf("Starting server on port %s\n", serverPort)
	if err := router.Run(serverPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}