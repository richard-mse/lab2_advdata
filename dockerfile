FROM golang:latest

WORKDIR /app

ADD graph/ graph/
ADD models/ models/

COPY go.mod go.sum main.go entrypoint.sh ./

RUN go mod tidy && \
chmod +x entrypoint.sh

ENTRYPOINT ["go", "run", "."]