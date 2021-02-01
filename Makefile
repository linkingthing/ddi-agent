GOSRC = $(shell find . -type f -name '*.go')

VERSION=v1.3.1

build: ddi_agent

ddi_agent: $(GOSRC) 
	CGO_ENABLED=0 GOOS=linux go build -o ddi_agent cmd/agent/agent.go

build-image:
	docker build -t linkingthing/ddi-agent:${VERSION} . 
	docker image prune -f

clean:
	rm -rf ddi_agent

.PHONY: clean install
