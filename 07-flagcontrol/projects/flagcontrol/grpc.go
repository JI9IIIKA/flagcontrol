package flagcontrol

import (
	"context"
	"crypto/tls"
	"encoding/json"
	p "flagcontrol/internal/platform"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"log/slog"
	"net"
	"os"
)

// ConfigService uses protobuf Struct to keep the example build independent of protoc.
// The checked-in proto file is the authoritative RPC contract.
type ConfigService interface {
	Get(context.Context, *structpb.Struct) (*structpb.Struct, error)
}
type configServer struct{ app *p.App }

func (s *configServer) Get(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	auth := md.Get("authorization")
	if len(auth) != 1 {
		return nil, status.Error(codes.Unauthenticated, "bearer token required")
	}
	token := auth[0]
	if len(token) < 7 || token[:7] != "Bearer " {
		return nil, status.Error(codes.Unauthenticated, "bearer token required")
	}
	principal, ok := s.app.Keys[token[7:]]
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	fields := in.GetFields()
	key := fields["key"].GetStringValue()
	env := fields["environment"].GetStringValue()
	if key == "" || env == "" {
		return nil, status.Error(codes.InvalidArgument, "key and environment required")
	}
	var raw []byte
	e := s.app.Pool.QueryRow(ctx, `SELECT row_to_json(f) FROM flags f JOIN flag_heads h USING(tenant,key,environment,version) WHERE tenant=$1 AND key=$2 AND environment=$3`, principal.Tenant, key, env).Scan(&raw)
	if e != nil {
		return nil, status.Error(codes.NotFound, "configuration unavailable")
	}
	var m map[string]any
	if e = json.Unmarshal(raw, &m); e != nil {
		return nil, status.Error(codes.Internal, "invalid configuration")
	}
	return structpb.NewStruct(m)
}
func rpcHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(structpb.Struct)
	if e := dec(in); e != nil {
		return nil, e
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(ConfigService).Get(ctx, req.(*structpb.Struct))
	}
	if interceptor == nil {
		return handler(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/portfolio.flags.Config/Get"}, handler)
}
func RegisterGRPC(a *p.App) {
	a.Background = append(a.Background, func(ctx context.Context) {
		addr := os.Getenv("GRPC_ADDR")
		if addr == "" {
			return
		}
		listener, e := net.Listen("tcp", addr)
		if e != nil {
			slog.Error("grpc listen failed", "error", e)
			return
		}
		defer listener.Close()
		options := []grpc.ServerOption{grpc.MaxRecvMsgSize(65536)}
		if cert := os.Getenv("GRPC_TLS_CERT"); cert != "" {
			pair, e := tls.LoadX509KeyPair(cert, os.Getenv("GRPC_TLS_KEY"))
			if e != nil {
				slog.Error("grpc TLS failed", "error", e)
				return
			}
			options = append(options, grpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12})))
		}
		server := grpc.NewServer(options...)
		desc := grpc.ServiceDesc{ServiceName: "portfolio.flags.Config", HandlerType: (*ConfigService)(nil), Methods: []grpc.MethodDesc{{MethodName: "Get", Handler: rpcHandler}}}
		server.RegisterService(&desc, &configServer{a})
		go func() { <-ctx.Done(); server.Stop() }()
		if e = server.Serve(listener); e != nil {
			slog.Error("grpc stopped", "error", e)
		}
	})
}
