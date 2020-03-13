FROM golang:latest

EXPOSE 8080
EXPOSE 8081

WORKDIR /proxy

COPY . .

RUN go mod download


CMD go run cmd/proxy/proxy.go  & go run cmd/repeater/repeater.go