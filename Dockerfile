FROM golang:1.24

WORKDIR /bot

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -v -o app .

CMD ["./app"]
