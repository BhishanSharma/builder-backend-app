package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

func ConnectDB() {
    godotenv.Load()
    mongoURI := os.Getenv("MONGODB_URI")
    
    // Set client options
    clientOptions := options.Client().ApplyURI(mongoURI)
    
    // Connect to MongoDB
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        log.Fatal("Failed to connect to MongoDB:", err)
    }
    
    err = client.Ping(ctx, nil)
    if err != nil {
        log.Fatal("Failed to ping MongoDB:", err)
    }
    
    fmt.Println("✅ Connected to MongoDB!")
    
    DB = client.Database("builder_db")
}

func GetCollection(collectionName string) *mongo.Collection {
    return DB.Collection(collectionName)
}