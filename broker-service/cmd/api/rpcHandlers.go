package main

import (
	"fmt"
	"net/http"
	"net/rpc"
)

const RPC_PORT = "5001"

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) logEventViaRPC(w http.ResponseWriter, logPayload LogPayload) {
	rpcClient, err := rpc.Dial("tcp", fmt.Sprintf("logger-service:%s", RPC_PORT))
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer rpcClient.Close()

	// define rpcPayload that RPC server is expecting
	rpcPayload := RPCPayload{
		Name: logPayload.Name,
		Data: logPayload.Data,
	}

	// call the service using RPC payload
	var result string
	err = rpcClient.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// create a JSON response, send it back to client using HTTP request (JSON)
	jsonPayload := JSONResponse{
		Error:   false,
		Message: result,
	}
	app.writeJSON(w, http.StatusAccepted, jsonPayload)
}
