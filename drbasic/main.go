package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/bmeg/protoc-gen-grcp-rest-direct/drtest"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type BasicServer struct {
	drtest.UnimplementedDirectServiceServer
}

func (bs *BasicServer) QueryGet(ctx context.Context, in *drtest.InputMessage) (*drtest.OutputMessage, error) {
	return &drtest.OutputMessage{Message: in.Message}, nil
}

func (bs *BasicServer) QueryPost(ctx context.Context, in *drtest.InputMessage) (*drtest.OutputMessage, error) {
	return &drtest.OutputMessage{Message: in.Message}, nil
}

func (bs *BasicServer) QueryStreamOut(in *drtest.InputMessage, srv drtest.DirectService_QueryStreamOutServer) error {
	for i := 0; i < 500; i++ {
		srv.Send(&drtest.OutputMessage{Message: fmt.Sprintf("%s : %d", in.Message, i)})
	}
	return nil
}

func (bs *BasicServer) QueryStreamIn(srv drtest.DirectService_QueryStreamInServer) error {
	//fmt.Printf("Starting input stream\n")
	count := 0
	for {
		msg, err := srv.Recv()
		if err == io.EOF {
			break
		}
		_ = msg
		//fmt.Printf("Input Stream got: %s\n", msg.Message)
		count++
	}
	//fmt.Printf("in stream closing: %d\n", count)
	return srv.SendAndClose(&drtest.OutputMessage{Message: fmt.Sprintf("count : %d", count)})
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &BasicServer{}

	lis, err := net.Listen("tcp", ":10002")
	if err != nil {
		fmt.Printf("Cannot open port: %v", err)
		return
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				unaryAuthInterceptor(),
				unaryInterceptor(),
			),
		),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				streamAuthInterceptor(),
				streamInterceptor(),
			),
		),
		grpc.MaxSendMsgSize(1024*1024*16),
		grpc.MaxRecvMsgSize(1024*1024*16),
	)

	// Regsiter Query Service
	drtest.RegisterDirectServiceServer(grpcServer, server)

	marsh := runtime.JSONPb{
		//protojson.MarshalOptions{EmitUnpopulated: true},
		//protojson.UnmarshalOptions{},
		//EnumsAsInts:  false,
		//EmitDefaults: true,
		//OrigName:     true,
	}

	grpcMux := runtime.NewServeMux(runtime.WithMarshalerOption("*/*", &marsh))

	err = drtest.RegisterDirectServiceHandlerClient(ctx, grpcMux,
		drtest.NewDirectServiceDirectClient(server,
			drtest.DirectUnaryInterceptor(unaryAuthInterceptor()),
			drtest.DirectStreamInterceptor(streamAuthInterceptor()),
		))
	if err != nil {
		fmt.Printf("registering query endpoint: %v", err)
		return
	}

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: grpcMux,
	}

	go func() {
		grpcServer.Serve(lis)
		cancel()
	}()

	go func() {
		httpServer.ListenAndServe()
		cancel()
	}()

	<-ctx.Done() //This will hold until canceled, usually from kill signal

}

// Return a new interceptor function that authorizes RPCs
// using a password stored in the config.
func unaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}

// Return a new interceptor function that authorizes RPCs
// using a password stored in the config.
func streamAuthInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}

// Check the context's metadata for the configured server/API password.
func authorize(ctx context.Context, user, password string) error {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md["authorization"]) > 0 {
			raw := md["authorization"][0]
			requser, reqpass, ok := parseBasicAuth(raw)
			if ok {
				if requser == user && reqpass == password {
					return nil
				}
				return grpc.Errorf(codes.PermissionDenied, "")
			}
		}
	}

	return grpc.Errorf(codes.Unauthenticated, "")
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
//
// Taken from Go core: https://golang.org/src/net/http/request.go?s=27379:27445#L828
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "

	if !strings.HasPrefix(auth, prefix) {
		return
	}

	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}

	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}

	return cs[:s], cs[s+1:], true
}

// Return a new interceptor function that logs all requests at the Info level
func unaryInterceptor() grpc.UnaryServerInterceptor {
	// Return a function that is the interceptor.
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		return resp, err
	}
}

// Return a new interceptor function that logs all requests at the Info level
// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/6f8030a0b4ee588a3f33556266b552a90a5574e2/logging/logrus/payload_interceptors.go#L46
func streamInterceptor() grpc.StreamServerInterceptor {
	// Return a function that is the interceptor.
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}
