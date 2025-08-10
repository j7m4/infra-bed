FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git gcc g++ musl-dev librdkafka-dev pkgconf

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod tidy

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -o go-spikes ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates librdkafka

WORKDIR /root/

COPY --from=builder /app/go-spikes .

EXPOSE 8080 6060

CMD ["./go-spikes"]