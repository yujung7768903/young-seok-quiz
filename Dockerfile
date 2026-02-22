# ── Build stage ──────────────────────────────────
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o server .

# ── Run stage ────────────────────────────────────
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY static/ ./static/
EXPOSE 8080
CMD ["./server"]
