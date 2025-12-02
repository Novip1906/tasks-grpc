FROM golang:1.25.1-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -v -o ./app ./cmd/gateway/main.go

CMD ["./app"]