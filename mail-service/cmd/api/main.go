package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Config struct {
	Mailer Mail
}

const (
	WEB_PORT = "80"
)

func main() {
	app := Config{
		Mailer: initMailer(),
	}

	log.Println("Starting mail service on port", WEB_PORT)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", WEB_PORT),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

// Mailer is used to send emails from microservice to users, that's why it only has FROM
func initMailer() Mail {
	port, _ := strconv.Atoi(os.Getenv("MAIL_PORT"))
	m := Mail{
		// Need to specify if using SendGrid or MailHog, better to use env for portability
		// TODO: Will add to docker-compose.yml
		Domain:      os.Getenv("MAIL_DOMAIN"),
		Host:        os.Getenv("MAIL_HOST"),
		Port:        port,
		Username:    os.Getenv("MAIL_USERNAME"),
		Password:    os.Getenv("MAIL_PASSWORD"),
		Encryption:  os.Getenv("MAIL_ENCRYPTION"),
		FromName:    os.Getenv("FROM_NAME"),
		FromAddress: os.Getenv("FROM_ADDRESS"),
	}
	return m
}
