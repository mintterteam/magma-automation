
FROM golang:1.18-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY amboss/ ./amboss/
COPY lnd/ ./lnd/
RUN ls -lrt
RUN go mod tidy
RUN go install

FROM alpine:latest
COPY --from=builder /go/bin/magma-automation /usr/local/bin/magma
CMD ["/usr/local/bin/magma"]