# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=latest
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-X main.version=$VERSION" \
    -o /app/terse ./cmd/terse

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/terse .

# Expose port (adjust if your app uses a different port)
EXPOSE 5000 

CMD ["./terse"]
