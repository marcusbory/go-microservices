package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

const FRONTEND_PORT = "8081" // remains unchanged

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		render(w, "test.page.gohtml")
	})

	fmt.Printf("Starting front end service on port %s\n", FRONTEND_PORT)
	err := http.ListenAndServe(fmt.Sprintf(":%s", FRONTEND_PORT), nil)
	if err != nil {
		log.Panic(err)
	}
}

func render(w http.ResponseWriter, t string) {

	partials := []string{
		"./cmd/web/templates/base.layout.gohtml",
		"./cmd/web/templates/header.partial.gohtml",
		"./cmd/web/templates/footer.partial.gohtml",
	}

	var templateSlice []string
	templateSlice = append(templateSlice, fmt.Sprintf("./cmd/web/templates/%s", t))

	for _, x := range partials {
		templateSlice = append(templateSlice, x)
	}

	tmpl, err := template.ParseFiles(templateSlice...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data struct {
		BrokerURL string
	}
	data.BrokerURL = os.Getenv("BROKER_URL")
	// ? Below is for testing / local dev only
	// data.BrokerURL = "http://localhost:80" // 80 is the port used by broker service

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
