FROM golang:1.25-bookworm AS builder
ENV GONOPROXY=
ENV CGO_ENABLED=0

# Install build dependencies for zencode-exec
RUN apt-get update && apt-get install -y build-essential cmake git wget

# Download or build zencode-exec based on architecture
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        wget -O /usr/local/zenroom-zencode-exec \
            "https://github.com/dyne/zenroom/releases/latest/download/zencode-exec" && \
        chmod +x /usr/local/zenroom-zencode-exec; \
    else \
        echo "Building zencode-exec from source for $ARCH..." && \
        git clone --depth 1 https://github.com/dyne/Zenroom.git /tmp/zenroom && \
        cd /tmp/zenroom && \
        make linux-zencode-exec && \
        cp zencode-exec /usr/local/zenroom-zencode-exec && \
        chmod +x /usr/local/zenroom-zencode-exec; \
    fi

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
ENV PATH="/usr/local/zenroom/bin:${PATH}"

RUN mkdir -p /data /usr/local/zenroom/bin

EXPOSE 8081

COPY --from=builder /interfacer-feedback-service /usr/local/bin/interfacer-feedback-service
COPY --from=builder /usr/local/zenroom-zencode-exec /usr/local/zenroom/bin/zencode-exec

CMD ["/usr/local/bin/interfacer-feedback-service"]
