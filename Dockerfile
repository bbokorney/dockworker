FROM golang:1.6

RUN mkdir /certs
ADD . /go/src/github.com/bbokorney/dockworker
WORKDIR /go/src/github.com/bbokorney/dockworker
ENV GOPATH=/go:/go/src/github.com/bbokorney/dockworker/Godeps/_workspace
RUN go build -o /app ./server

CMD ["/app"]
