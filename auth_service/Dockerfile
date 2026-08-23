FROM golang:alpine AS builder

WORKDIR /app
COPY common/ /app/common/
COPY auth_service/go.mod auth_service/go.sum ./auth_service/
WORKDIR /app/auth_service
RUN go mod download

COPY auth_service/ .
RUN go build -o main ./cmd/api/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auth_service/main .
COPY --from=builder /app/auth_service/migrations ./migrations

CMD ["./main"]
