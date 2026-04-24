package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type RequestPayload struct {
	// What action wants to be done
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth"`

	// TODO: Add other payloads here next time (Log, Mail, AMQP)
	Log  LogPayload  `json:"log"`
	Mail MailPayload `json:"mail"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := JSONResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// handle the request payload accordingly
	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, r, requestPayload.Auth)
	case "log":
		// ? Instead of logging via logger service, we will be logging via RabbitMQ
		app.logEventViaRabbit(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)
	default:
		app.errorJSON(w, errors.New("unknown action"), http.StatusBadRequest)
		return
	}
}

// ? New function handle logging written in rabbitHandlers.go
// ! These functions will be deprecated because we will be using RabbitMQ (queue system to process asynchronously)

// For sending mail
func (app *Config) sendMail(w http.ResponseWriter, m MailPayload) {
	// create some json that we'll send to the mail microservice
	jsonData, _ := json.MarshalIndent(m, "", "\t")

	// call the service - whatever we named in our docker-compose.yml + route we defined in the mail service
	request, err := http.NewRequest("POST", "http://mail-service/send", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling mail service"))
		return
	}

	// create a variable we'll read response.Body into
	var payload JSONResponse
	payload.Error = false
	payload.Message = "Email sent to " + m.To
	app.writeJSON(w, http.StatusAccepted, payload)
}

// For logging
func (app *Config) logItem(w http.ResponseWriter, l LogPayload) {
	// create some json that we'll send to the logger microservice
	jsonData, _ := json.MarshalIndent(l, "", "\t")

	// call the service - whatever we named in our docker-compose.yml + route we defined in the logger service
	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling logger service"))
		return
	}

	// create a variable we'll read response.Body into
	var payload JSONResponse
	payload.Error = false
	payload.Message = "Logged!"
	app.writeJSON(w, http.StatusAccepted, payload)
}

// For authentication
func (app *Config) authenticate(w http.ResponseWriter, r *http.Request, a AuthPayload) {
	// create some json that we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	// call the service - whatever we named in our docker-compose.yml + route we defined in the authentication service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// make sure we get back the correct status code
	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		app.errorJSON(w, err)
		return
	}
	// read the response body, don't leave it open
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		// Provide status code for unauthorized
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	} else if response.StatusCode != http.StatusAccepted {
		// Provide status code for accepted
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService JSONResponse

	// decode the response body into jsonFromService
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	// write the response to client
	var payload JSONResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = jsonFromService.Data
	app.writeJSON(w, http.StatusAccepted, payload)
}
