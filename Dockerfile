FROM golang:1.8-alpine
ADD . /go/src/github.com/kumina/dovecot_exporter
WORKDIR /go/src/github.com/kumina/dovecot_exporter
RUN apk add --no-cache git && \
    go get -v ./... && \
    go build

FROM alpine:latest
EXPOSE 9166
WORKDIR /
COPY --from=0 /go/src/github.com/kumina/dovecot_exporter/dovecot_exporter .
ENTRYPOINT ["/dovecot_exporter"]
