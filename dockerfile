FROM golang:latest

WORKDIR /app

ADD graph/ graph/
ADD models/ models/
ADD data/ data/

COPY go.mod go.sum main.go ./
RUN go mod tidy

ENTRYPOINT ["go", "run", "."]

CMD [""]