from golang:1.5.3-alpine

RUN mkdir -p /var/www
COPY ./client/www/app/template/ /var/www/
ADD . /go/src/github.com/sdcoffey/olympus
RUN go install github.com/sdcoffey/olympus/server
ENTRYPOINT /go/bin/server
