#!/usr/bin/env bash
go install github.com/google/go-licenses@latest
go mod download
go-licenses report ./... --template hack/attribution.tmpl > ATTRIBUTION.md

