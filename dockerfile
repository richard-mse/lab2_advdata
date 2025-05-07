FROM golang:latest

WORKDIR /app

ADD graph/ graph/
ADD models/ models/

COPY go.mod go.sum main.go ./
RUN go mod tidy

ENTRYPOINT ["go", "run", "."]

CMD ["--limit=1000"]