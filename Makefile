
# Build

build-prod osmanage:
	CGO_ENABLED=0 go build -a -ldflags="-s -w" -o osmanage ./cmd/osmanage

build-dev:
	go build -o osmanage ./cmd/osmanage

gen-proto:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	export PATH="$$PATH:$$(go env GOPATH)/bin"; \
	protoc --go_out=. --go_opt=module=github.com/OpenSlides/openslides-cli --go-grpc_out=. --go-grpc_opt=module=github.com/OpenSlides/openslides-cli proto/osmanage.proto

# Tests / Linting

test:
	go test -v -count=1 -race -shuffle=on -coverprofile=coverage.txt ./...

format:
	gofmt -s -w . && git diff --exit-code

