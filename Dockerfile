ARG GO_VERSION=1.24.2
FROM golang:${GO_VERSION} 
WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]
