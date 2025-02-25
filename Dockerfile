# Stage 1: Build the Go binary using a Go Alpine image
FROM golang:1.24-alpine AS builder

WORKDIR /app
# Copy all project files into the container
COPY . .
# Build the binary with CGO disabled for a static build
RUN CGO_ENABLED=0 go build -o filesentry .

# Stage 2: Create a minimal runtime image using Alpine
FROM alpine:latest
WORKDIR /app
# Copy the binary from the builder stage
COPY --from=builder /app/filesentry .
# Create the directory that will be watched
RUN mkdir -p /app/watcher
# Set the command to run your binary
CMD ["./filesentry"]
