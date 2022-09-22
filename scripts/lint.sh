#!/bin/bash

ROOT=$(cd `dirname $0`/.. && pwd)
cd $ROOT

# 执行go内置代码审查工具
go vet ./...

# 此工具的功能已经被golangci-lint所包含
#go install honnef.co/go/tools/cmd/staticcheck@latest
#staticcheck ./...

# 执行开源代码审查工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2
golangci-lint run ./...
