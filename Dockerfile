FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git && \
    mkdir -p $GOPATH/src/github.com/benkillin/GolangFactionsBot/

WORKDIR $GOPATH/src/github.com/benkillin/GolangFactionsBot/

COPY .git .
COPY vendor .
COPY EmbedHelper .
COPY FactionsBot.go .
COPY FactionsBot_test.go .
COPY factionsBotConfig.json .

#WORKDIR $GOPATH/src/github.com/benkillin/
#RUN go get -d -v github.com/benkillin/GolangFactionsBot
#WORKDIR $GOPATH/src/github.com/benkillin/GolangFactionsBot/
RUN go build -o /go/bin/FactionsBot FactionsBot.go

FROM scratch

RUN mkdir -p /opt/FactionsBot/ && \
    mkdir -p /opt/FactionsBot/bin/ && \
    mkdir -p /opt/FactionsBot/logs/

COPY --from=builder /go/bin/FactionsBot /opt/FactionsBot/bin/

WORKDIR /opt/FactionsBot/

CMD /opt/FactionsBot/bin/FactionsBot/
