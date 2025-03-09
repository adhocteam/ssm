#!/usr/bin/env bash
set -euxo pipefail
rm -rf release
mkdir release
GOOS=darwin GOARCH=amd64 go build -o release/ssm-darwin-amd64
GOOS=linux GOARCH=amd64 go build -o release/ssm-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o release/ssm-darwin-arm64
cd release
sha256sum ssm-darwin-amd64 >ssm-darwin-amd64.sha
sha256sum ssm-darwin-amd64 >ssm-linux-amd64.sha
sha256sum ssm-darwin-arm64 >ssm-darwin-arm64.sha
