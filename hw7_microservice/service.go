package main

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
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
		auth[consumer] = methods
	}
	a = &aclAuth{
		acl: auth,
	}

	return
}

type eventLogs struct {
	id int
	e  map[int]chan *Event

	mu *sync.Mutex
}

func newEventLogs() *eventLogs {
	e := &eventLogs{}
	e.e = make(map[int]chan *Event)

	e.mu = &sync.Mutex{}

	return e
}

func (e *eventLogs) NewChan() (chan *Event, int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.id++
	e.e[e.id] = make(chan *Event)

	return e.e[e.id], e.id
}

func (e *eventLogs) Notify(evt *Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, c := range e.e {
		c <- evt
	}
}

func (e *eventLogs) Delete(id int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	close(e.e[id])
	delete(e.e, id)
}

func (e *eventLogs) DeleteAll() {
	for id := range e.e {
		e.Delete(id)
	}
}

type adminServer struct {
	UnimplementedAdminServer

	e *eventLogs
}

func newAdminServer(e *eventLogs) (as adminServer) {
	as = adminServer{}
	as.e = e
	return
}

func (as adminServer) Logging(_ *Nothing, server Admin_LoggingServer) error {
	ch, id := as.e.NewChan()
	defer as.e.Delete(id)

	for c := range ch {
		err := server.Send(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (as adminServer) Statistics(interval *StatInterval, server Admin_StatisticsServer) error {
	return nil
}

type middleWare struct {
	auth *aclAuth
	opts []grpc.ServerOption

	e *eventLogs
}

func newMiddleWare(a *aclAuth, e *eventLogs) (mw *middleWare, err error) {
	err = nil
	mw = &middleWare{
		auth: a,
	}

	mw.opts = []grpc.ServerOption{
		grpc.UnaryInterceptor(mw.unaryInterceptor),
		grpc.StreamInterceptor(mw.streamInterceptor),
	}

	mw.e = e

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

	host := ""
	if p, ok := peer.FromContext(ctx); ok {
		host = p.Addr.String()
	}

	e := &Event{
		Consumer: consumers[0],
		Method:   method,
		Host:     host,
	}

	mw.e.Notify(e)

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

	e := newEventLogs()

	mw, err := newMiddleWare(auth, e)
	if err != nil {
		return
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return
	}

	server := grpc.NewServer(mw.opts...)

	RegisterAdminServer(server, newAdminServer(e))
	RegisterBizServer(server, newBizServer())

	go server.Serve(lis)
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	return
}
