# Stage 1: Build the Go binaries
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum (if they exist) to cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source files
COPY generate_token.go .
COPY auth-server/ ./auth-server/

# Build generate_token binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/generate_token generate_token.go

# Build auth-server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/auth_server ./auth-server/main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/generate_token /app/generate_token
COPY --from=builder /app/auth_server /app/auth_server
COPY entrypoint.sh /app/entrypoint.sh

# Copy config.yml
COPY config.yml /app/config.yml
#
# Copy users.json
COPY /users.json /app/users.json

# Ensure binaries are executable
RUN chmod +x /app/generate_token /app/auth_server /app/entrypoint.sh

# Expose port 4222 (NATS default, if auth-server uses it)
EXPOSE 4222

# Set default entrypoint to run auth-server
ENTRYPOINT ["/app/entrypoint.sh"]
