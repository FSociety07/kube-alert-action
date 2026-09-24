
FROM golang:1.26.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o controller .


FROM alpine:3.22

RUN addgroup -S operator && adduser -S operator -G operator

WORKDIR /app

COPY --from=builder /app/controller /app/controller

USER operator

ENTRYPOINT ["/app/controller"]