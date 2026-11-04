# --- Stage 1: Build (Builder) ---
FROM golang:alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-server ./cmd/api

# --- Stage 2: Final runtime image ---
FROM alpine:latest

# Install CA certificates and timezone data for secure AWS HTTPS calls
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -u 10001 appuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/api-server .

# Run as non-root user
USER appuser

EXPOSE 8080

CMD ["./api-server"]
