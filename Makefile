TAG?=latest
VERSION?=$(shell grep 'VERSION' cmd/kjob/main.go | awk '{ print $$4 }' | tr -d '"')

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

release:
	git tag "v$(VERSION)"
	git push origin "v$(VERSION)"