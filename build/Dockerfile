FROM golang:latest
RUN go get -u github.com/kardianos/govendor  \
    && mkdir -p $GOPATH/src/github.com/bocheninc/L0 \
    && cd $GOPATH/src/github.com/bocheninc/L0 \
    && wget http://192.168.8.109:81/L0-master.tar.gz -O L0.tar.gz   \
    && tar -xf L0.tar.gz  \
    && mv L0*/* . \
    && go install -tags=embed ./cmd/lcnd/...

CMD ["lcnd"]