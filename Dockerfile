# Dependabot tracks both the version and digest, including CA bundle updates.
FROM golang:1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test ./... && go vet ./...

RUN go build -trimpath -mod=readonly -o /out/main .
RUN go test -c -tags=container -o /out/container-tests .

FROM scratch AS runtime

# Use the distribution's trust store without adding certificate code to the app.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/main /app/main
COPY --from=builder /src/LICENSE /LICENSE

USER 65532:65532
WORKDIR /app
CMD ["/app/main"]

FROM runtime AS container-test
COPY --from=builder /out/container-tests /app/container-tests
CMD ["/app/container-tests", "-test.v", "-test.run=^TestContainer", "-test.timeout=60s"]

# Keep the default build free of the test executable.
FROM runtime AS final
