FROM node:20-alpine AS frontend
WORKDIR /app
COPY web/app/package*.json ./
RUN npm ci
COPY web/app/ ./
RUN npm run build

FROM golang:1.26-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /internal/server/dashboard_dist ./internal/server/dashboard_dist/
RUN CGO_ENABLED=1 go build -o /peerclawd ./cmd/peerclawd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite-libs
COPY --from=builder /peerclawd /usr/local/bin/peerclawd
COPY configs/peerclaw.example.yaml /etc/peerclaw/config.yaml
EXPOSE 8080
ENTRYPOINT ["peerclawd"]
CMD ["-config", "/etc/peerclaw/config.yaml"]
