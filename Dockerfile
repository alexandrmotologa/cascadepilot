# Stage 1: Build React 19 Frontend
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Reverse Proxy Binary
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy compiled web assets into server/dist for go:embed
COPY --from=web-builder /app/internal/server/dist ./internal/server/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/cascadepilot ./cmd/cascadepilot

# Stage 3: Minimal Runtime Container
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/bin/cascadepilot /usr/local/bin/cascadepilot
COPY examples/ /app/examples/

EXPOSE 8088
VOLUME ["/app/data"]

ENTRYPOINT ["cascadepilot"]
CMD ["serve", "--config", "examples/cascadepilot.yaml"]
