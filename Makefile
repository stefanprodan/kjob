TAG?=latest
VERSION?=$(shell grep 'VERSION' pkg/version/version.go | awk '{ print $$4 }' | tr -d '"')
LT_VERSION?=$(shell grep 'VERSION' cmd/loadtester/main.go | awk '{ print $$4 }' | tr -d '"' | head -n1)

build:
	CGO_ENABLED=0 go build -o ./bin/kjob ./cmd/kjob

fmt:
	gofmt -l -s -w ./
	goimports -l -w ./

test-fmt:
	gofmt -l -s ./ | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi
	goimports -l ./ | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

test:
	go test ./...

