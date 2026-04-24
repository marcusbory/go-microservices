package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"broker/logs"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GRPC_PORT = "50001"

func (app *Config) logEventViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// create a new client using new code
	log.Println("Dialing Logger Service")
	conn, err := grpc.NewClient(fmt.Sprintf("logger-service:%s", GRPC_PORT),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	defer conn.Close()

	client := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// ? Code is written at logger-service/cmd/api/grpc.go
	// Write log via gRPC
	_, err = client.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := JSONResponse{
		Error:   false,
		Message: "logged via gRPC",
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
