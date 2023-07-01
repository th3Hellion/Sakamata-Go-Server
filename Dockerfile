FROM golang:1.20-alpine

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o app

EXPOSE 80

CMD ["./app"]
