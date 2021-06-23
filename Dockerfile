FROM golang:1.16.5-alpine3.13 AS BUILD
LABEL MAINTAINER="Replica <yumeko@outlook.com>"

# make binary
RUN apk add --no-cache ca-certificates curl make alpine-sdk linux-headers
WORKDIR /go/src/github.com/projecteru2/barrel
COPY . /go/src/github.com/projecteru2/barrel/
RUN make build && ./eru-barrel --version

FROM alpine:3.11
LABEL MAINTAINER="Replica <yumeko@outlook.com>"

RUN mkdir /etc/eru/
COPY --from=BUILD /go/src/github.com/projecteru2/barrel/eru-barrel /usr/bin/eru-barrel
COPY barrel.conf /etc/eru/