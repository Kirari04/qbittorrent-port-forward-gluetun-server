# Dependabot tracks both the version and digest, including CA bundle updates.
FROM golang:1.27.2-alpine3.24@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder

WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go test ./... && go vet ./...
RUN go build -trimpath -mod=readonly -o /out/main .

FROM scratch AS runtime

# Use the distribution's trust store without adding certificate code to the app.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/main /app/main

USER 65532:65532
WORKDIR /app
CMD ["/app/main"]

# Run the actual application in its runtime filesystem during container tests.
FROM builder AS test-builder
RUN go test -c -tags=container -o /out/container-tests .

FROM runtime AS container-test
COPY --from=test-builder /out/container-tests /app/container-tests
CMD ["/app/container-tests", "-test.v", "-test.run=^TestContainer", "-test.timeout=60s"]

# Keep the default build free of the test executable.
FROM runtime AS final
