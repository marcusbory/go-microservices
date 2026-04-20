# Go Microservices Project

Building a frontend that connects to 5 microservices
1. Broker (optional entry point to microservices)
2. Auth (Postgres)
3. Logger (MongoDB)
4. Mailer (sends email with a template)
5. Listener (consumes messages in RabbitMQ and initiates a process)

Communication is done via:
- REST API with JSON as transport
- Send and receive RPC (remote procedure call)
- Send and recive gRPC (Google's high performance implementation of RPC model)
- Initiate and respond to events using AMQP (advanced message queue protocol)

Personal Notes
---

#### Port Usage

| Port | Service        |
| ---- | -------------- |
| 8080 | Broker         |
| 8081 | Frontend       |
| 8082 | Authentication |


> **Credits:**  
This project structure and examples are based on the course ["Go Microservices"](https://github.com/tsawler/go-microservices) by [@tsawler](https://github.com/tsawler).  