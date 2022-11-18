
# protoc-gen-grcp-rest-direct

A protoc code generator that creates stub wrappers to directly connect
grpc http gateway code to GRPC services.

The [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) creates a
JSON HTTP server that wraps a GRPC based service. It acts as a reverse proxy and
connects using a GRPC client to a second GRPC based port. This means that the
user request via HTTP is:
 1. received as JSON
 2. Is deserialized to native GO structure
 3. Passed to GRPC client
 4. Serialized using protobuf
 5. Send over network GRPC port
 6. Deserialized from protobuf to native GO structure
 7. Passed to server function

`protoc-gen-grcp-rest-direct` skips most of that process by building a direct
client/server connection stub, to that the grpc-gateway HTTP server can directly
call the GRPC server methods, without going through the additional
serialize/network/deserialize steps.

## Limitations
 - Doesn't map Streaming Input/Output (but neither does GRPC Gateway)
 - Limited server options (has unary/stream interceptors)
