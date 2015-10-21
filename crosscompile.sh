#!/bin/bash

for target in darwin:amd64 linux:amd64 linux:386 linux:arm windows:amd64; do
  echo "Compiling $target"
  export GOOS=$(echo $target | cut -d: -f1) GOARCH=$(echo $target | cut -d: -f2)
  bash -c "go build -o $(basename $(echo $PWD))_${GOOS}_${GOARCH} ."
done
