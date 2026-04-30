# base go image
FROM golang:1.25.0-alpine AS builder

# create a directory for the application
RUN mkdir /app

# copy the application code to the container
COPY . /app

# set the working directory
WORKDIR /app

# build the application into the mailApp binary
RUN CGO_ENABLED=0 go build -o frontApp ./cmd/web

RUN chmod +x /app/frontApp

# build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/frontApp /app

CMD ["/app/frontApp"]


