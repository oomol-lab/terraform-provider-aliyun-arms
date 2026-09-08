default: fmt test build generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmtcheck:
	@test -z "$$(gofmt -l -s .)" || (gofmt -l -s .; exit 1)

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt fmtcheck lint test testacc build install generate
