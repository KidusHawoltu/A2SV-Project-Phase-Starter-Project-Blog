# --- Stage 1: Build ---
# Use the official Golang image as a builder.
# Using a specific version ensures consistent builds.
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies.
COPY go.mod go.sum ./
# Download dependencies. This is cached as a separate layer, so it only re-runs if dependencies change.
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Build the Go application.
# -o /app/server specifies the output path for the binary.
# CGO_ENABLED=0 is important for creating a static binary that doesn't depend on system C libraries.
# -ldflags="-w -s" strips debugging information, making the binary smaller.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./Delivery/main.go


# --- Stage 2: Final Image ---
# "alpine" is a good alternative if you need a shell for debugging.
FROM alpine:latest

# Set the working directory.
WORKDIR /app

# --- THIS IS THE CRITICAL PART ---
# Copy the Certificate Authority certificates from the builder stage.
# This ensures your application can verify TLS connections.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary from the 'builder' stage.
COPY --from=builder /app/server .

# Expose the port the application will run on.
# This should match the PORT in your .env file or config.
EXPOSE 8080

# The command to run when the container starts.
ENTRYPOINT ["/app/server"]