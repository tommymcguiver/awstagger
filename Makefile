build:
	mkdir build

.PHONY: build/awstagger
build/awstagger:
	GO111MODULE=on go build -o build/awstagger

.PHONY: test
test:
	GO111MODULE=on go test ./...
