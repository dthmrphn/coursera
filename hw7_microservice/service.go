package main

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

type bizServer struct {
	UnimplementedBizServer
}

func newBizServer() (bs bizServer) {
	bs = bizServer{}
	return
}

func (bs bizServer) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (bs bizServer) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (bs bizServer) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

type aclAuth struct {
	acl map[string][]string
}

func newAclAuth(acl string) (a *aclAuth, err error) {
	paths := make(map[string][]string)
	err = json.Unmarshal([]byte(acl), &paths)
	if err != nil {
		return
	}

	auth := make(map[string][]string, len(paths))
	for consumer, methods := range paths {
		// auth[consumer] = make([]string, len(methods))
		// for _, method := range methods {
		// 	auth[consumer] = strings.Split(method, "/")
		// }
		auth[consumer] = methods
	}
	a = &aclAuth{
		acl: auth,
	}

	return
}

type adminServer struct {
	UnimplementedAdminServer
}

func newAdminServer() (as adminServer) {
	as = adminServer{}
	return
}

func (as adminServer) Logging(_ *Nothing, server Admin_LoggingServer) error {
	return nil
}

func (as adminServer) Statistics(interval *StatInterval, server Admin_StatisticsServer) error {
	return nil
}

type middleWare struct {
	auth *aclAuth
	opts []grpc.ServerOption
}

func newMiddleWare(a *aclAuth) (mw *middleWare, err error) {
	err = nil
	mw = &middleWare{
		auth: a,
	}

	mw.opts = []grpc.ServerOption{
		grpc.UnaryInterceptor(mw.unaryInterceptor),
		grpc.StreamInterceptor(mw.streamInterceptor),
	}

	return
}

func (mw *middleWare) authorize(ctx context.Context, method string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "failed to get metadata")
	}

	consumers, ok := md["consumer"]
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unknown consumer")
	}

	allowedMethods, ok := mw.auth.acl[consumers[0]]
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unknown consumer")
	}

	isAllowed := false
	for _, m := range allowedMethods {
		if m == method || strings.Split(m, "/")[2] == "*" {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return status.Errorf(codes.Unauthenticated, "method isnt allowed")
	}

	return nil
}

func (mw *middleWare) invoke(ctx context.Context, method string) error {
	return nil
}

func (mw *middleWare) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	err := mw.authorize(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (mw *middleWare) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := mw.authorize(ss.Context(), info.FullMethod)
	if err != nil {
		return err
	}
	return handler(srv, ss)
}

func StartMyMicroservice(ctx context.Context, listenAddr, ACLData string) (err error) {
	auth, err := newAclAuth(ACLData)
	if err != nil {
		return
	}

	mw, err := newMiddleWare(auth)
	if err != nil {
		return
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return
	}

	server := grpc.NewServer(mw.opts...)

	RegisterAdminServer(server, newAdminServer())
	RegisterBizServer(server, newBizServer())

	go server.Serve(lis)
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	return
}
