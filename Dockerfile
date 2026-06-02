FROM golang:1.25-bookworm AS builder
ENV GONOPROXY=
ENV CGO_ENABLED=0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /interfacer-feedback-service ./cmd/main

FROM dyne/devuan:chimaera
WORKDIR /root

ENV HOST=0.0.0.0
ENV PORT=8081
ENV GIN_MODE=release
ENV SQLITE_PATH=/data/feedback.db

RUN mkdir -p /data

EXPOSE 8081

COPY --from=builder /interfacer-feedback-service /usr/local/bin/interfacer-feedback-service

CMD ["/usr/local/bin/interfacer-feedback-service"]
