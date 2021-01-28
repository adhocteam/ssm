#!/usr/bin/env bash
set -euxo pipefail
rm -rf release
mkdir release
GOOS=darwin GOARCH=amd64 go build -o ssm-darwin-amd64
sha256sum ssm-darwin-amd64 >ssm-darwin-amd64.sha
GOOS=linux GOARCH=amd64 go build -o ssm-linux-amd64
sha256sum ssm-darwin-amd64 >ssm-linux-amd64.sha
