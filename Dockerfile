FROM golang:onbuild
#FROM golang:1.6.2-alpine

#RUN apk update && \
#    apk upgrade && \
#    apk add git bash bash-completion

ENV LOC /go/src/github.com/GregorioMartinez/redditbot

ADD . $LOC

WORKDIR $LOC

#RUN go get
RUN go install

# Copy over blacklists
RUN cp *.txt /go/bin

ENTRYPOINT /go/bin/redditbot