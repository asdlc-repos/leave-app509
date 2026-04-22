# syntax=docker/dockerfile:1.6

FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dispatcher ./

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -u 10001 app
USER app
WORKDIR /app

COPY --from=builder /out/dispatcher /app/dispatcher

EXPOSE 9090
ENTRYPOINT ["/app/dispatcher"]
