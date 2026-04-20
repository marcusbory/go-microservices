package main

import (
	"authentication/data" // module name is authentication (no service)
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	// import the postgres driver, but don't use it directly
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

// Docker lets multiple services listen to the same port (80)
const WEB_PORT = "80"

const MAX_RETRIES = 20

var counts int64

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println("Starting authentication service")

	// connect to DB
	conn, connErr := connectToDB()
	if connErr != nil || conn == nil {
		log.Panic("Can't connect to Postgres!")
	}
	defer conn.Close()

	// init config
	app := Config{
		DB:     conn,
		Models: data.New(conn),
	}

	// listen for web connections
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", WEB_PORT),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func connectToDB() (*sql.DB, error) {
	connectionString := os.Getenv("DSN")

	for {
		// stay there until connection is successful
		connection, err := openDB(connectionString)
		if err != nil {
			log.Println("Postgres not yet ready...")
			counts++
		} else {
			log.Println("Connected to Postgres!")
			return connection, nil
		}

		// break the loop if we've tried MAX_RETRIES times (~40s)
		if counts > MAX_RETRIES {
			log.Println(err)
			return nil, err
		}

		log.Println("Backing off for 2 seconds...")
		time.Sleep(2 * time.Second)
	}
}
