#!/usr/bin/env bash

set -exuo pipefail

# A simple helper script. There's probably something more proper that I'll replace this with later.

go fmt *.go
go mod tidy
go run .
