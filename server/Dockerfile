# syntax=docker/dockerfile:1

# ── Build stage ──────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Static binary: the runtime stage has no libc toolchain.
# Migrations are embedded via go:embed (see migrations.go).
RUN CGO_ENABLED=0 go build -o /out/nori .

# ── Runtime stage ────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=build /out/nori ./nori

EXPOSE 8080
CMD ["./nori", "serve"]
