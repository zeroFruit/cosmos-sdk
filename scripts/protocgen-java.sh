#!/usr/bin/env bash

set -eo pipefail
mkdir -p build/java

proto_dirs=$(find ./ -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  protoc -I "proto" -I "third_party/proto" -I "testutil/testdata" --java_out=build/java $(find "${dir}" -maxdepth 1 -name '*.proto')

done
