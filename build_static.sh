#!/bin/sh

docker run -i -v `pwd`:/dovecot_exporter alpine:3.14 /bin/sh << 'EOF'
set -ex

# Install prerequisites for the build process.
apk update
apk add ca-certificates git go libc-dev
update-ca-certificates

# Build the dovecot_exporter.
cd /dovecot_exporter
export GOPATH=/gopath
export GO111MODULE=auto
go get -d ./...
go build --ldflags '-extldflags "-static"'
strip dovecot_exporter
EOF
