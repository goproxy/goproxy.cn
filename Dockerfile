FROM golang:1.20-alpine3.18 AS build

WORKDIR /usr/local/src/goproxy.cn
COPY . .

RUN apk add --no-cache git
RUN go mod download
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/

FROM alpine:3.18

COPY --from=build /usr/local/src/goproxy.cn/bin/ /usr/local/bin/
COPY templates/ /goproxy.cn/templates/
COPY assets/ /goproxy.cn/assets/
COPY locales/ /goproxy.cn/locales/
COPY robots.txt favicon.ico apple-touch-icon.png unknown-badge.svg /goproxy.cn/

RUN apk add --no-cache go git git-lfs openssh gpg subversion fossil mercurial breezy
RUN git lfs install

ENV GOPATH=/tmp/gopath
ENV GOCACHE=/tmp/gocache
ENV GOPROXY=direct
ENV GOSUMDB=off

WORKDIR /goproxy.cn

ENTRYPOINT ["/usr/local/bin/goproxy.cn"]
