GOSRC = $(shell find . -type f -name '*.go')

build: ddi_agent

ddi_agent: $(GOSRC) 
	CGO_ENABLED=0 GOOS=linux go build -o ddi_agent cmd/agent/agent.go

clean:
	rm -rf ddi_agent

.PHONY: clean install
