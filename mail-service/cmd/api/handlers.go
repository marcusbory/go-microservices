package main

import "net/http"

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	type mailMessage struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	var requestPayload mailMessage

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	msg := Message{
		From:    requestPayload.From,
		To:      requestPayload.To,
		Subject: requestPayload.Subject,
		Data:    requestPayload.Message,
	}

	// Add to app config!
	err = app.Mailer.SendSMTPMessage(msg)
	if err != nil {
		app.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	payload := JSONResponse{
		Error:   false,
		Message: "Email sent to " + requestPayload.To,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
