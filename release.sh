#!/usr/bin/env bash
GOOS=darwin GOARCH=amd64 go build -o ssm-darwin-amd64
sha256sum ssm-darwin-amd64 >ssm-darwin-amd64.sha
GOOS=linux GOARCH=amd64 go build -o ssm-linux-amd64
sha256sum ssm-darwin-amd64 >ssm-linux-amd64.sha
