# base go image
FROM golang:1.25.0-alpine AS builder

# create a directory for the application
RUN mkdir /app

# copy the application code to the container
COPY . /app

# set the working directory
WORKDIR /app

# build the application into the loggerApp binary
RUN CGO_ENABLED=0 go build -o loggerApp ./cmd/api

RUN chmod +x /app/loggerApp

# build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/loggerApp /app

CMD ["/app/loggerApp"]
