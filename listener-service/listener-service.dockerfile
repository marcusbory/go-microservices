# base go image
FROM golang:1.25.0-alpine AS builder

# create a directory for the application
RUN mkdir /app

# copy the application code to the container
COPY . /app

# set the working directory
WORKDIR /app

# build the application into the listenerApp binary
# ! take note, no cmd/api folder because we're not using a web server, so we're building the main.go file
RUN CGO_ENABLED=0 go build -o listenerApp ./main.go

RUN chmod +x /app/listenerApp

# build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app/listenerApp /app

CMD ["/app/listenerApp"]
