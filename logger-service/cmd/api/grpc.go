package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"log-service/logs"
	"net"

	"google.golang.org/grpc"
)

type LogServer struct {
	// ensure backwards compatibility with old RPC server
	logs.UnimplementedLogServiceServer
	Models data.Models
}

func (l *LogServer) WriteLog(ctx context.Context, req *logs.LogRequest) (*logs.LogResponse, error) {
	input := req.GetLogEntry()

	// write the log using generated code from logs.proto, received via gRPC
	logEntry := data.LogEntry{
		Name: input.Name,
		Data: input.Data,
	}

	// insert log entry into MongoDB
	err := l.Models.LogEntry.Insert(logEntry)
	if err != nil {
		return &logs.LogResponse{Result: "Failed to log"}, err
	}

	return &logs.LogResponse{Result: "Logged"}, nil
}

func (app *Config) gRPCListen() {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%s", GRPC_PORT))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v\n", err)
	}

	srv := grpc.NewServer()
	logs.RegisterLogServiceServer(srv, &LogServer{Models: app.Models})

	log.Printf("gRPC server started on port %s\n", GRPC_PORT)

	if err := srv.Serve(listen); err != nil {
		log.Fatalf("Failed to serve (gRPC server): %v\n", err)
	}
}
