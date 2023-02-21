build: generate
	go build ./cmd/eks-node-viewer

download:
	go mod download
	go mod tidy

licenses: download
	go-licenses check ./... --allowed_licenses=MIT,Apache-2.0,BSD-3-Clause,ISC \
	--ignore github.com/mattn/go-localereader # MIT

boilerplate:
	go run hack/boilerplate.go ./

verify: boilerplate licenses download
	gofmt -w -s ./.
	golangci-lint run


coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out


generate:
	# run generate twice, gen_licenses needs the ATTRIBUTION file or it fails.  The second run
	# ensures that the latest copy is embedded when we build.
	go generate ./...
	./hack/gen_licenses.sh
	go generate ./...
.PHONY: verify boilerplate licenses download coverage generate
