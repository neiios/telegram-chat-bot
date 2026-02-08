FROM golang:1.25 AS builder

COPY --from=sqlc/sqlc:latest /workspace/sqlc /usr/local/bin/sqlc

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN sqlc generate
RUN CGO_ENABLED=0 go build -o /bot .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /bot /bot

ENTRYPOINT ["/bot"]
