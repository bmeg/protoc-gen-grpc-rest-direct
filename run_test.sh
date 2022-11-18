#!/bin/bash


if [ ! -e bin ]; then
    mkdir bin
fi

go build ./

mv protoc-gen-grcp-rest-direct bin/

if [ ! -e ./bin/protoc-gen-go ]; then
    GOBIN=`pwd`/bin/ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if [ ! -e ./bin/protoc-gen-go-grpc ]; then
    GOBIN=`pwd`/bin/ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

if [ ! -e ./bin/protoc-gen-grpc-gateway ]; then
    GOBIN=`pwd`/bin/ go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
fi


export PATH=`pwd`/bin:$PATH

pushd drtest/ 

protoc \
    -I ./ \
	-I ../googleapis/ \
	--go_out . \
	--go_opt paths=source_relative \
	--go-grpc_out ./ \
	--go-grpc_opt paths=source_relative \
    --grpc-gateway_out ./ \
    --grpc-gateway_opt logtostderr=true \
    --grpc-gateway_opt paths=source_relative \
    --grcp-rest-direct_out . \
	basic.proto

popd

go build -o test/ ./drbasic

