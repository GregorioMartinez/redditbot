FROM golang:onbuild

ENV LOC /go/src/github.com/GregorioMartinez/redditbot

ADD . $LOC

WORKDIR $LOC

RUN go install


ENTRYPOINT /go/bin/redditbot