#!/bin/sh

set -e

go build main.go
cp main ~/go/bin/renderstatic