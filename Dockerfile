# Use the official Golang image as the base
FROM golang:1.25-alpine AS builder

# Set environment variables
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory inside the container
WORKDIR /build

# Copy the entire application source
COPY . .

# Build the Go binary
RUN go build -o /app .

# Final lightweight stage
FROM alpine:3.21 AS final

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app /app/server
COPY static/ /app/static/

# Expose the application's port
EXPOSE 8090

# Run the application
CMD ["/app/server"]
