# syntax=docker/dockerfile:1
FROM golang:1.24.0-alpine3.21

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-speedtest
ENV GIN_MODE=release

EXPOSE 8080

# Run
CMD ["/go-speedtest"]