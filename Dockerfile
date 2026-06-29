FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/subscription-service ./cmd/server

FROM alpine:3.21

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/subscription-service /app/subscription-service
COPY migrations /app/migrations
COPY docs /app/docs
USER app

EXPOSE 8080
ENTRYPOINT ["/app/subscription-service"]
