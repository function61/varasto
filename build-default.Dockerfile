FROM fn61/buildkit-golang:20181202_1219_65241f45f27f49b2

WORKDIR /go/src/github.com/function61/bup

CMD bin/build.sh

RUN ln -s /go/src/github.com/function61/bup/rel/bup_linux-amd64 /usr/local/bin/bup
