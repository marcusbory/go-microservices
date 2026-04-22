package main

import (
	"context"
	"time"

	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	WEB_PORT  = "80"
	RPC_PORT  = "5001"
	MONGO_URI = "mongodb://mongo:27017"
	GRPC_PORT = "50001"
)

var client *mongo.Client

type Config struct {
}

func main() {
	// connect to MongoDB
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	defer mongoClient.Disconnect(context.TODO())

	// set client
	client = mongoClient

	// create a context in order to disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// close connection (when program exits)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}

func connectToMongo() (*mongo.Client, error) {
	// create connection options
	clientOptions := options.Client().ApplyURI(MONGO_URI)
	clientOptions.SetAuth(options.Credential{
		Username: "admin",
		Password: "password",
	})

	// connect to MongoDB
	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println("Error connecting to MongoDB:", err)
		return nil, err
	}

	// check the connection
	err = c.Ping(context.TODO(), nil)
	if err != nil {
		log.Println("Error pinging MongoDB:", err)
		return nil, err
	}

	return c, nil
}
