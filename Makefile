
# Build

build-prod osmanage:
	CGO_ENABLED=0 go build -a -ldflags="-s -w" -o osmanage ./cmd/osmanage

build-dev:
	go build -o osmanage ./cmd/osmanage


# Tests / Linting

test:
	go test -v -count=1 -race -shuffle=on -coverprofile=coverage.txt ./...

format:
	gofmt -s -w . && git diff --exit-code

