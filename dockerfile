FROM golang:latest

WORKDIR /app

ADD graph/ graph/
ADD models/ models/

ADD http://vmrum.isc.heia-fr.ch/biggertest.json data/biggertest.json

COPY go.mod go.sum main.go ./
RUN go mod tidy

ENTRYPOINT ["go", "run", "."]

CMD [""]