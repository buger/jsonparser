FROM golang:1.6

RUN go get github.com/Jeffail/gabs
RUN go get github.com/bitly/go-simplejson
RUN go get github.com/pquerna/ffjson
RUN go get github.com/antonholmquist/jason

WORKDIR /go/src/github.com/buger/jsonparser
ADD . /go/src/github.com/buger/jsonparser