.PHONY: help clean verify boilerplate licenses download coverage generate

NO_COLOR=\033[0m
GREEN=\033[32;01m
YELLOW=\033[33;01m
RED=\033[31;01m

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[33m%-20s\033[0m %s\n", $$1, $$2}'

build: generate ## Build
	go build -ldflags="-s -w -X main.version=local -X main.builtBy=Makefile" ./cmd/eks-node-viewer

goreleaser: ## Release snapshot
	goreleaser build --snapshot --rm-dist

download: ## Download dependencies
	go mod download
	go mod tidy

licenses: download ## Check licenses
	go-licenses check ./... --allowed_licenses=MIT,Apache-2.0,BSD-3-Clause,ISC \
	--ignore github.com/mattn/go-localereader # MIT

boilerplate: ## Add license headers
	go run hack/boilerplate.go ./

verify: boilerplate licenses download ## Format and Lint
	gofmt -w -s ./.
	golangci-lint run

coverage: ## Run tests w/ coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

generate: ## Generate attribution
	# run generate twice, gen_licenses needs the ATTRIBUTION file or it fails.  The second run
	# ensures that the latest copy is embedded when we build.
	go generate ./...
	./hack/gen_licenses.sh
	go generate ./...

clean: ## Clean artifacts
	rm -rf eks-node-viewer
	rm -rf dist/
