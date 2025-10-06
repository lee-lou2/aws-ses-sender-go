FROM golang:1.25-alpine AS builder

WORKDIR /usr/src/app

# Install build dependencies for CGO (Alpine uses musl libc)
RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build with CGO enabled for SQLite support
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o app .

FROM alpine:3.21

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

COPY --from=builder /usr/src/app/.env .env
COPY --from=builder /usr/src/app/app .

CMD ["./app"]
