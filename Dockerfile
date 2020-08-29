FROM golang:1.14.5-alpine3.12 AS build

ENV GOPROXY=https://goproxy.io

RUN mkdir -p /go/src/github.com/linkingthing/ddi-agent
COPY . /go/src/github.com/linkingthing/ddi-agent

WORKDIR /go/src/github.com/linkingthing/ddi-agent
RUN CGO_ENABLED=0 GOOS=linux go build -o ddi-agent cmd/agent/agent.go

FROM alpine:3.12
COPY --from=build /go/src/github.com/linkingthing/ddi-agent/ddi-agent /
ENTRYPOINT ["/ddi-agent"]
