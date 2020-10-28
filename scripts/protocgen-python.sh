#!/usr/bin/env bash

set -eo pipefail
mkdir -p build/python

proto_dirs=$(find ./ -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  protoc -I "proto" -I "third_party/proto" -I "testutil/testdata" --python_out=build/python $(find "${dir}" -maxdepth 1 -name '*.proto')

done
