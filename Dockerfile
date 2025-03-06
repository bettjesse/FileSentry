# Stage 1: Build the Go binary using a Go Alpine image
FROM golang:1.24-alpine AS builder

WORKDIR /app
# Copy all project files into the container
COPY . .
# Build the binary from the cmd/filesentry folder
RUN CGO_ENABLED=0 go build -o filesentry ./cmd/filesentry

# Stage 2: Create a minimal runtime image using Alpine
FROM alpine:latest
WORKDIR /app
# Copy the binary from the builder stage
COPY --from=builder /app/filesentry .
# Create the directory that will be watched
RUN mkdir -p /app/data/watcher
# Ensure the configs folder exists and is accessible if needed
COPY ./configs /app/configs
# Set the command to run your binary
CMD ["./filesentry"]
