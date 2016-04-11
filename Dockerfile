FROM golang:1.6

RUN mkdir /certs
ADD . /go/src/github.com/bbokorney/dockworker
WORKDIR /go/src/github.com/bbokorney/dockworker
RUN go get -v
RUN go build -o /app ./server

CMD ["/app"]
