# Stage 1: Build the Go application
FROM golang:1.24 AS builder
WORKDIR /app

# Copy go module files and download dependencies first to leverage cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application using the argument
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gform .
# If your main package is not in the root, use:
# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o your_app_name ./cmd/server/  # Example path

FROM alpine:latest
WORKDIR /root/


# Copy the built binary from the builder stage using the argument
COPY --from=builder /app/gform .

# Copy any necessary static assets or configuration files if needed
# COPY --from=builder /app/config.yaml .
# COPY --from=builder /app/static ./static

# Expose the port the Go application listens on (e.g., 8080)
EXPOSE 8080

# Command to run the executable using the argument
CMD ["./gform"]
