#!/usr/bin/env bash

# see https://golang.org/doc/install/source#environment
targets="linux:386"

sha=$(git rev-parse HEAD)
for t in "$targets"; do
  os=$(echo "$t" | awk '{split($0,a,":"); print a[1]}')
  arch=$(echo "$t" | awk '{split($0,a,":"); print a[2]}')
  GOOS="$os" GOARCH="$arch" go build -o "lidder-$sha-$os-$arch" lidder.go
done

