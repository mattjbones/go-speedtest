# syntax=docker/dockerfile:1
FROM golang:1.24.0-alpine3.21 AS build

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-speedtest

FROM alpine:3.21.3 AS main 

ENV GIN_MODE=release
EXPOSE 8080

COPY --from=build /go-speedtest /

# Run
CMD ["/go-speedtest"]