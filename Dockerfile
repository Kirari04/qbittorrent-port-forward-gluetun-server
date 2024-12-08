FROM golang:alpine AS builder

WORKDIR /app

# Copy the Go module files
COPY go.mod ./
COPY go.sum ./

# Download dependencies (optional, but recommended for caching)
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o main .

# Use a smaller base image for the final image
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Set the entrypoint to the Go application
CMD ["./main"]