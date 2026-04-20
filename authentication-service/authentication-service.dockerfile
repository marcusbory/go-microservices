# base go imago
FROM golang:1.25.0-alpine AS builder

# create a directory for the application
RUN mkdir /app

# copy the application code to the container
COPY . /app

# set the working directory
WORKDIR /app

# build the application into the authenticationApp binary
RUN CGO_ENABLED=0 go build -o authenticationApp ./cmd/api

RUN chmod +x /app/authenticationApp

# build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/authenticationApp /app

CMD ["/app/authenticationApp"]
