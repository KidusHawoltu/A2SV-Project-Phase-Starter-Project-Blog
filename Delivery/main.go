package main

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	"A2SV_Starter_Project_Blog/Delivery/routers"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// --- 0. Load Configuration / Environment Variables ---
	err := godotenv.Load("../.env")
	if err != nil {
		// if not in parent folder, try to load from current folder
		err = godotenv.Load()
		if err != nil {
			log.Printf("Warning: No .env file found or failed to load: %v", err)
		}
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
		log.Printf("Warning: MONGO_URI not set, using default '%s'\n", mongoURI)
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "g6_blog_db"
		log.Printf("Warning: DB_NAME not set, using default '%s'\n", dbName)
	}

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// --- 1. Initialize External Resources (MongoDB connection) ---
	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err = mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("Warning: Failed to disconnect from MongoDB: %v", err)
		}
	}()

	// Ping the primary to verify that the connection is alive.
	if err = mongoClient.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Fatal: Failed to ping MongoDB: %v", err)
	}
	log.Println("MongoDB connection established successfully.")

	// Select the database and collection
	db := mongoClient.Database(dbName)
	blogCollection := db.Collection("blogs")

	// --- 2. Instantiate Concrete Repository Implementations ---
	blogRepository := repositories.NewBlogRepository(blogCollection)
	log.Println("Repositories initialized.")
	// (When you add users, the user repository would be initialized here)

	// --- 3. Instantiate Usecases ---
	// Define a timeout for usecase context deadlines
	usecaseTimeout := 5 * time.Second
	blogUsecase := usecases.NewBlogUsecase(blogRepository, usecaseTimeout)
	log.Println("Usecases initialized.")
	// (When you add users, the user usecase would be initialized here)

	// --- 4. Instantiate Delivery Controllers ---
	blogController := controllers.NewBlogController(blogUsecase)
	log.Println("Controllers initialized.")
	// (Auth middleware would be initialized here when you add it)

	// --- 5. Set Up Delivery Routers ---
	router := gin.Default()

	// You can add global middlewares here like CORS, logging, etc.
	// router.Use(middlewares.CORSMiddleware())

	// We pass the main router engine to our setup functions.
	routers.SetupBlogRouter(router, blogController)
	// (When you add users, you would call routers.SetupUserRouter here)

	log.Println("All Routers configured.")

	// --- 6. Start the HTTP Server ---
	log.Printf("Server starting on port %s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Fatal: Server failed to start: %v", err)
	}
}
