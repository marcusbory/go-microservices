package main

import (
	"fmt"
	"log"
	"net/http"
)

// Docker can listen to port 80 for any content
const WEB_PORT = "80"

type Config struct {
}

func main() {
	app := Config{}

	log.Printf("Starting broker service on port %s\n", WEB_PORT)

	// define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", WEB_PORT),
		Handler: app.routes(),
	}

	// start the server
	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}
