#!/usr/bin/env bash
go install github.com/google/go-licenses@latest
go mod download
GOROOT=$(go env GOROOT) go-licenses report ./... --template hack/attribution.tmpl > ATTRIBUTION.md

