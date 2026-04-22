# base go image
FROM golang:1.25.0-alpine AS builder

# create a directory for the application
RUN mkdir /app

# copy the application code to the container
COPY . /app

# set the working directory
WORKDIR /app

# build the application into the mailApp binary
RUN CGO_ENABLED=0 go build -o mailApp ./cmd/api

RUN chmod +x /app/mailApp

# build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/mailApp /app

# copy the templates to the container
COPY templates /templates

CMD ["/app/mailApp"]

