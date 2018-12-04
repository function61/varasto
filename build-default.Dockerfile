FROM fn61/buildkit-golang:20181204_1302_5eedb86addc826e7

WORKDIR /go/src/github.com/function61/bup

CMD bin/build.sh

RUN ln -s /go/src/github.com/function61/bup/rel/bup_linux-amd64 /usr/local/bin/bup
