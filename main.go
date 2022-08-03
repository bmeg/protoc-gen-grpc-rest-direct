package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

var CODE_HEADER string = `
// Code generated by protoc-gen-grpc-rest-direct. DO NOT EDIT.
package {{.Package}}

import (
	"io"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// use io module, incase generated sections don't to avoid 'import not used' error
var _ = io.EOF 

type DirectOption func(directClientBase)

type directClientBase interface {
  setUnaryInterceptor(grpc.UnaryServerInterceptor)
  setStreamInterceptor(grpc.StreamServerInterceptor)
}

func DirectUnaryInterceptor(g grpc.UnaryServerInterceptor) func(directClientBase) {
  return func(d directClientBase) {
    d.setUnaryInterceptor(g)
  }
}

func DirectStreamInterceptor(g grpc.StreamServerInterceptor) func(directClientBase) {
  return func(d directClientBase) {
    d.setStreamInterceptor(g)
  }
}

`

var CODE_SERVICE string = `
// {{.Service}}DirectClient is a shim to connect {{.Service}} client directly server
type {{.Service}}DirectClient struct {
  unaryServerInt grpc.UnaryServerInterceptor
  streamServerInt grpc.StreamServerInterceptor
	server {{.Service}}Server
}
 // New{{.Service}}DirectClient creates new {{.Service}}DirectClient
func New{{.Service}}DirectClient(server {{.Service}}Server, opts ...DirectOption) *{{.Service}}DirectClient {
	o := &{{.Service}}DirectClient{server:server}
  for _, opt := range opts {
    opt(o)
  }
  return o
}

func (shim *{{.Service}}DirectClient) setUnaryInterceptor(a grpc.UnaryServerInterceptor) {
  shim.unaryServerInt = a
}

func (shim *{{.Service}}DirectClient) setStreamInterceptor(a grpc.StreamServerInterceptor) {
  shim.streamServerInt = a
}

`

var CODE_SHIM string = `{{if .StreamOutput}}

/* Start {{.Service}}{{.Name}} call output server  */
type direct{{.Service}}{{.Name}} struct {
  ctx context.Context
  c   chan *{{.OutputType}}
  e   error
}

func (dsm *direct{{.Service}}{{.Name}}) Recv() (*{{.OutputType}}, error) {
	value, ok := <-dsm.c
	if !ok {
    if dsm.e != nil {
      return nil, dsm.e
    }
		return nil, io.EOF
	}
	return value, dsm.e
}

func (dsm *direct{{.Service}}{{.Name}}) Send(a *{{.OutputType}}) error {
	return dsm.SendMsg(a)
}

func (dsm *direct{{.Service}}{{.Name}}) SendMsg(m interface{}) error  { 
	dsm.c <- m.(*{{.OutputType}})
	return nil 
}

func (dsm *direct{{.Service}}{{.Name}}) close() {
	close(dsm.c)
}
func (dsm *direct{{.Service}}{{.Name}}) Context() context.Context {
	return dsm.ctx
}
func (dsm *direct{{.Service}}{{.Name}}) CloseSend() error             { return nil }
func (dsm *direct{{.Service}}{{.Name}}) SetTrailer(metadata.MD)       {}
func (dsm *direct{{.Service}}{{.Name}}) SetHeader(metadata.MD) error  { return nil }
func (dsm *direct{{.Service}}{{.Name}}) SendHeader(metadata.MD) error { return nil }
func (dsm *direct{{.Service}}{{.Name}}) RecvMsg(m interface{}) error  { return nil }
func (dsm *direct{{.Service}}{{.Name}}) Header() (metadata.MD, error) { return nil, nil }
func (dsm *direct{{.Service}}{{.Name}}) Trailer() metadata.MD         { return nil }
/* End {{.Service}}{{.Name}} call output server  */

func (shim *{{.Service}}DirectClient) {{.Name}}(ctx context.Context, in *{{.InputType}}, opts ...grpc.CallOption) ({{.Service}}_{{.Name}}Client, error) {
  md, _ := metadata.FromOutgoingContext(ctx)
  ictx := metadata.NewIncomingContext(ctx, md)

	w := &direct{{.Service}}{{.Name}}{ictx, make(chan *{{.OutputType}}, 100), nil}
  if shim.streamServerInt != nil {
    go func() {
      defer w.close()
      info := grpc.StreamServerInfo{
        FullMethod: "/{{.Package}}.{{.Service}}/{{.Name}}",
        IsServerStream: true,
      }
      shim.streamServerInt(shim.server, w, &info, _{{.Service}}_{{.Name}}_Handler)
    } ()
    return w, nil
  }
	go func() {
    defer w.close()
		w.e = shim.server.{{.Name}}(in, w)
	}()
	return w, nil
}
{{else if .StreamInput}}
// Streaming data 'server' shim. Provides the Send/Recv funcs expected by the
// user server code when dealing with a streaming input

/* Start {{.Service}}{{.Name}} streaming input server */
type direct{{.Service}}{{.Name}} struct {
  ctx context.Context
  c   chan *{{.InputType}}
  out chan *{{.OutputType}}
}

func (dsm *direct{{.Service}}{{.Name}}) Recv() (*{{.InputType}}, error) {
	value, ok := <-dsm.c
	if !ok {
		return nil, io.EOF
	}
	return value, nil
}

func (dsm *direct{{.Service}}{{.Name}}) Send(a *{{.InputType}}) error {
	dsm.c <- a
	return nil
}

func (dsm *direct{{.Service}}{{.Name}}) Context() context.Context {
	return dsm.ctx
}

func (dsm *direct{{.Service}}{{.Name}}) SendAndClose(o *{{.OutputType}}) error {
  dsm.out <- o
  close(dsm.out)
  return nil
}

func (dsm *direct{{.Service}}{{.Name}}) CloseAndRecv() (*{{.OutputType}}, error) {
  //close(dsm.c)
  out := <- dsm.out
  return out, nil
}

func (dsm *direct{{.Service}}{{.Name}}) CloseSend() error             { close(dsm.c); return nil }
func (dsm *direct{{.Service}}{{.Name}}) SetTrailer(metadata.MD)       {}
func (dsm *direct{{.Service}}{{.Name}}) SetHeader(metadata.MD) error  { return nil }
func (dsm *direct{{.Service}}{{.Name}}) SendHeader(metadata.MD) error { return nil }
func (dsm *direct{{.Service}}{{.Name}}) SendMsg(m interface{}) error  { dsm.out <- m.(*{{.OutputType}}); return nil }

func (dsm *direct{{.Service}}{{.Name}}) RecvMsg(m interface{}) error  { 
	t, err := dsm.Recv()
	mPtr := m.(*{{.InputType}}) 
	if t != nil {
    	*mPtr = *t
	}
	return err
}

func (dsm *direct{{.Service}}{{.Name}}) Header() (metadata.MD, error) { return nil, nil }
func (dsm *direct{{.Service}}{{.Name}}) Trailer() metadata.MD         { return nil }
/* End {{.Service}}{{.Name}} streaming input server */


func (shim *{{.Service}}DirectClient) {{.Name}}(ctx context.Context, opts ...grpc.CallOption) ({{.Service}}_{{.Name}}Client, error) {
  md, _ := metadata.FromOutgoingContext(ctx)
  ictx := metadata.NewIncomingContext(ctx, md)
  w := &direct{{.Service}}{{.Name}}{ictx, make(chan *{{.InputType}}, 100), make(chan *{{.OutputType}}, 3)}
  if shim.streamServerInt != nil {
    info := grpc.StreamServerInfo{
      FullMethod: "/{{.Package}}.{{.Service}}/{{.Name}}",
      IsClientStream: true,
    }
    go shim.streamServerInt(shim.server, w, &info, _{{.Service}}_{{.Name}}_Handler)
    return w, nil
  }
	go func() {
		shim.server.{{.Name}}(w)
	}()
	return w, nil
}
{{else}}
//{{.Name}} shim
func (shim *{{.Service}}DirectClient) {{.Name}}(ctx context.Context, in *{{.InputType}}, opts ...grpc.CallOption) (*{{.OutputType}}, error) {
  md, _ := metadata.FromOutgoingContext(ctx)
  ictx := metadata.NewIncomingContext(ctx, md)
  if shim.unaryServerInt != nil {
    handler := func(ctx context.Context, req interface{}) (interface{}, error) {
  		return shim.server.{{.Name}}(ctx, req.(*{{.InputType}}))
  	}
    info := grpc.UnaryServerInfo{
      FullMethod: "/{{.Package}}.{{.Service}}/{{.Name}}",
    }
    o, err := shim.unaryServerInt(ictx, in, &info, handler)
    if o == nil {
      return nil, err
    }
    return o.(*{{.OutputType}}), err
  }
	return shim.server.{{.Name}}(ictx, in)
}{{end}}
`

func contains(c []string, a string) bool {
	for _, i := range c {
		if a == i {
			return true
		}
	}
	return false
}

type headerDesc struct {
	Package string
}

type serviceDesc struct {
	Service string
}

type methodDesc struct {
	Package      string
	Service      string
	Name         string
	InputType    string
	OutputType   string
	StreamOutput bool
	StreamInput  bool
}

func cleanProtoType(name string, p string) string {
	if strings.HasPrefix(name, "."+p+".") {
		return name[len(p)+2:]
	}
	return name
}

func boolPtrDefaultFalse(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func main() {
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("failed to read code generator request: %v", err)
		return
	}
	req := new(plugin.CodeGeneratorRequest)
	if err = proto.Unmarshal(input, req); err != nil {
		log.Printf("failed to unmarshal code generator request: %v", err)
		return
	}

	headerTemplate, _ := template.New("header").Parse(CODE_HEADER)
	serviceTemplate, _ := template.New("service").Parse(CODE_SERVICE)
	shimTemplate, err := template.New("shim").Parse(CODE_SHIM)
	if err != nil {
		log.Fatal(err)
	}

	out := []*plugin.CodeGeneratorResponse_File{}
	for _, file := range req.ProtoFile {
		if contains(req.FileToGenerate, *file.Name) {
			//log.Printf("File: %s", *file.Name)
			text := bytes.NewBufferString("")
			headerTemplate.Execute(text, headerDesc{Package: *file.Package})
			for _, service := range file.Service {
				//log.Printf("Service: %s", *service.Name)
				serviceTemplate.Execute(text, serviceDesc{Service: *service.Name})
				for _, method := range service.Method {
					//log.Printf(" method: %s", method)
					err := shimTemplate.Execute(text, methodDesc{
						Package: *file.Package,
						Service: *service.Name, Name: *method.Name,
						InputType:    cleanProtoType(*method.InputType, *file.Package),
						OutputType:   cleanProtoType(*method.OutputType, *file.Package),
						StreamOutput: boolPtrDefaultFalse(method.ServerStreaming),
						StreamInput:  boolPtrDefaultFalse(method.ClientStreaming),
					})
					if err != nil {
						log.Printf("Error: %s", err)
					}
				}
			}
			n := strings.Replace(*file.Name, ".proto", ".pb.dgw.go", -1)
			t := text.String()
			f := &plugin.CodeGeneratorResponse_File{Name: &n, Content: &t}
			out = append(out, f)
		}
	}

	resp := &plugin.CodeGeneratorResponse{File: out}

	buf, err := proto.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		log.Fatal(err)
	}

}
