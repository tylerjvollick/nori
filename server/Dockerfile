# syntax=docker/dockerfile:1

FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o nori .

EXPOSE 8080
CMD ["./nori"]

