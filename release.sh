#!/usr/bin/env bash
set -euxo pipefail
rm -rf release
mkdir release
GOOS=darwin GOARCH=amd64 go build -o release/ssm-darwin-amd64
sha256sum release/ssm-darwin-amd64 >release/ssm-darwin-amd64.sha
GOOS=linux GOARCH=amd64 go build -o release/ssm-linux-amd64
sha256sum release/ssm-darwin-amd64 >release/ssm-linux-amd64.sha
