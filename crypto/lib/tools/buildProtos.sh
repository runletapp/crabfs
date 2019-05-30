#!/usr/bin/env bash

set -e

OUTPUT=./protos

mkdir -p $OUTPUT

# generate js codes via grpc-tools
grpc_tools_node_protoc \
--js_out=import_style=commonjs,binary:$OUTPUT \
--grpc_out=$OUTPUT \
--plugin=protoc-gen-grpc=`which grpc_tools_node_protoc_plugin` \
-I ./protos \
./protos/*.proto

# generate d.ts codes
protoc \
--plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
--ts_out=$OUTPUT \
-I ./protos \
./protos/*.proto