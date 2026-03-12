FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /incident-io-exporter .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /incident-io-exporter /incident-io-exporter

USER 65534:65534

EXPOSE 9090

ENTRYPOINT ["/incident-io-exporter"]
