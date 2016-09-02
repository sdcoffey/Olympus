FROM golang:1.7.0-alpine

ADD . /go/src/github.com/sdcoffey/olympus
RUN go install github.com/sdcoffey/olympus/server
ENTRYPOINT /go/bin/server
