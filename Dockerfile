FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pr-reviewer ./cmd/app

FROM alpine:3.19

WORKDIR /app

RUN adduser -D -g '' appuser

COPY --from=builder /app/pr-reviewer /app/pr-reviewer

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/pr-reviewer"]
