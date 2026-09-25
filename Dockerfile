ARG BUILDPLATFORM
FROM --platform=${BUILDPLATFORM} golang:1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# Copy the Go module files
COPY go.mod ./
COPY go.sum ./

# Download dependencies (optional, but recommended for caching)
RUN go mod download

# Copy the source code
COPY . .

RUN go test ./...

# Build the Go application
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} go build -o /app/main .

# Use a smaller base image for the final image
FROM scratch AS minimal

# Copy the built binary from the builder stage
COPY --from=builder /app/main /app/main
COPY --from=builder /app/LICENSE /LICENSE

USER 1000:1000

# Set the entrypoint to the Go application
CMD ["/app/main"]
