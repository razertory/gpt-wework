FROM golang:1.19-alpine

COPY . .

RUN go build .

CMD ["./gpt-wework"]