#!/bin/bash

ROOT=$(cd `dirname $0`/.. && pwd)
cd $ROOT

go test ./...
