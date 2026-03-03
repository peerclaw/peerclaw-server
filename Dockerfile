FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 go build -o /peerclawd ./cmd/peerclawd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=builder /peerclawd /usr/local/bin/peerclawd
COPY configs/peerclaw.example.yaml /etc/peerclaw/config.yaml

EXPOSE 8080 9090

ENTRYPOINT ["peerclawd"]
CMD ["-config", "/etc/peerclaw/config.yaml"]
