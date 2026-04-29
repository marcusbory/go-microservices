package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Docker can listen to port 80 for any content
const WEB_PORT = "80"

type Config struct {
	Rabbit *amqp.Connection
}

func main() {
	// try to connect to RabbitMQ
	rabbitConn, err := connect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	app := Config{Rabbit: rabbitConn}

	log.Printf("Starting broker service on port %s\n", WEB_PORT)

	// define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", WEB_PORT),
		Handler: app.routes(),
	}

	// start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connect() (*amqp.Connection, error) {
	// write back off calls
	var counts int64
	var connection *amqp.Connection

	// don't continue until RabbitMQ is ready
	for {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq-service")
		// Must match the Kubernetes Service DNS name for RabbitMQ.
		// Will change for docker-compose.yml
		if err != nil {
			fmt.Println("RabbitMQ not ready yet...")
			counts++
		} else {
			log.Println("Connected to RabbitMQ!")
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}

		backOff := time.Duration(math.Pow(float64(counts), 2)) * time.Second

		fmt.Println("Backing off...")
		time.Sleep(backOff)
		continue
	}

	return connection, nil
}
