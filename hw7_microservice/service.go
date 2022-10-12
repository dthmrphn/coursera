package main

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	"google.golang.org/grpc"
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
		auth[consumer] = make([]string, len(methods))
		for _, method := range methods {
			auth[consumer] = strings.Split(method, "/")
		}
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

func (as adminServer) Logging(_ *Nothing, alog Admin_LoggingServer) error {
	return nil
}

func (as adminServer) Statistics(*StatInterval, Admin_StatisticsServer) error {
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
		opts: []grpc.ServerOption{
			grpc.UnaryInterceptor(mw.unaryInterceptor),
			grpc.StreamInterceptor(mw.streamInterceptor),
		},
	}
	return
}

func (mw *middleWare) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	resp = nil
	err = nil
	return
}

func (mw *middleWare) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	err = nil
	return
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
