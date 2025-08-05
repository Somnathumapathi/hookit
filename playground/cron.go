// package main

// import (
// 	"sync"
// 	"time"

// 	"gofr.dev/pkg/gofr"
// )

// var (
// 	n  = 0
// 	mu sync.RWMutex
// )

// const duration = 3

// func main() {
// 	app := gofr.New()

// 	app.AddCronJob("* * * * * *", "counter", count)
// 	time.Sleep(duration * time.Second)
// }

// func count(c *gofr.Context) {
// 	mu.Lock()
// 	defer mu.Unlock()

// 	n++

//		c.Log("Count:", n)
//	}
package playground

import (
	"context"
	"database/sql"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func executeAction(step Step, input map[string]interface{}) error {
	// Extract file data from the input
	fileData, ok := input["file_data"].([]byte)
	if !ok {
		return fmt.Errorf("missing or invalid file data in input")
	}

	// Parse destination configuration from step payload
	destConfig, ok := step.Payload["destination"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing destination configuration in step payload")
	}

	// Determine the database type
	dbType, ok := destConfig["db_type"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid database type in destination configuration")
	}

	// Handle the destination based on the database type
	switch dbType {
	case "firebase":
		return uploadToFirebase(destConfig, fileData)
	case "mongodb":
		return uploadToMongo(destConfig, fileData)
	case "custom":
		return uploadToCustomDB(destConfig, fileData)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
}
func executeAction(step Step, input map[string]interface{}) error {
	// Extract file data from the input
	fileData, ok := input["file_data"].([]byte)
	if !ok {
		return fmt.Errorf("missing or invalid file data in input")
	}

	// Parse destination configuration from step payload
	destConfig, ok := step.Payload["destination"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing destination configuration in step payload")
	}

	// Determine the database type
	dbType, ok := destConfig["db_type"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid database type in destination configuration")
	}

	// Handle the destination based on the database type
	switch dbType {
	case "firebase":
		return uploadToFirebase(destConfig, fileData)
	case "mongodb":
		return uploadToMongo(destConfig, fileData)
	case "custom":
		return uploadToCustomDB(destConfig, fileData)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
}
func uploadToFirebase(config map[string]interface{}, fileData []byte) error {
	// Extract Firebase configuration
	storageBucket, ok := config["storage_bucket"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid Firebase storage bucket")
	}

	// Initialize Firebase client
	client, err := firebase.NewClient(storageBucket, config["credentials"].(string))
	if err != nil {
		return fmt.Errorf("failed to initialize Firebase client: %w", err)
	}

	// Upload file
	err = client.UploadFile("parsed_file.json", fileData)
	if err != nil {
		return fmt.Errorf("failed to upload file to Firebase: %w", err)
	}

	return nil
}
func uploadToMongo(config map[string]interface{}, fileData []byte) error {
	// Extract MongoDB configuration
	connectionString, ok := config["connection_string"].(string)
	if !ok {
		return fmt.Errorf("missing MongoDB connection string")
	}
	dbName, ok := config["database_name"].(string)
	if !ok {
		return fmt.Errorf("missing database name")
	}
	collectionName, ok := config["collection_name"].(string)
	if !ok {
		return fmt.Errorf("missing collection name")
	}

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(connectionString))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(context.TODO())

	// Insert file data
	collection := client.Database(dbName).Collection(collectionName)
	_, err = collection.InsertOne(context.TODO(), map[string]interface{}{
		"file": string(fileData),
	})
	if err != nil {
		return fmt.Errorf("failed to insert file into MongoDB: %w", err)
	}

	return nil
}
func uploadToCustomDB(config map[string]interface{}, fileData []byte) error {
	// Extract custom database configuration
	host, _ := config["host"].(string)
	port, _ := config["port"].(string)
	user, _ := config["user"].(string)
	password, _ := config["password"].(string)
	dbName, _ := config["db_name"].(string)

	// Build the DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, dbName)

	// Connect to the database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to custom database: %w", err)
	}
	defer db.Close()

	// Insert file data
	_, err = db.Exec("INSERT INTO files (data) VALUES (?)", string(fileData))
	if err != nil {
		return fmt.Errorf("failed to insert file into custom database: %w", err)
	}

	return nil
}
