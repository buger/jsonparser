FROM golang:1.10

ARG PACKAGES='\
    github.com/a8m/djson \
    github.com/antonholmquist/jason \
    github.com/bitly/go-simplejson \
    github.com/Jeffail/gabs \
    github.com/mailru/easyjson \
    github.com/mreiferson/go-ujson \
    github.com/pquerna/ffjson \
    github.com/ugorji/go/codec \
'

RUN go get $PACKAGES

WORKDIR /go/src/github.com/buger/jsonparser
ADD . /go/src/github.com/buger/jsonparser