package main

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type microservice struct {
	UnimplementedBizServer
	UnimplementedAdminServer
	acl   map[string][]string
	mu    sync.Mutex
	logs  map[*logSubscription]struct{}
	stats map[*Stat]struct{}
}

type logSubscription struct {
	events   chan *Event
	overflow chan struct{}
}

func StartMyMicroservice(ctx context.Context, addr string, acl string) error {
	s := &microservice{logs: make(map[*logSubscription]struct{}), stats: make(map[*Stat]struct{})}
	if err := json.Unmarshal([]byte(acl), &s.acl); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(s.unary), grpc.StreamInterceptor(s.stream))
	RegisterBizServer(server, s)
	RegisterAdminServer(server, s)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer listener.Close()
		server.Serve(listener)
	}()
	go func() {
		select {
		case <-ctx.Done():
			server.Stop()
		case <-done:
		}
	}()
	return nil
}

func (s *microservice) authorize(ctx context.Context, method string) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	consumers := md.Get("consumer")
	if len(consumers) == 1 && consumers[0] != "" {
		for _, allowed := range s.acl[consumers[0]] {
			if allowed == method || (strings.HasSuffix(allowed, "/*") && strings.HasPrefix(method, strings.TrimSuffix(allowed, "*"))) {
				return consumers[0], nil
			}
		}
	}
	return "", status.Error(codes.Unauthenticated, "consumer has no access to method")
}

// Record before dispatch, so existing subscribers see new admin calls as well.
func (s *microservice) record(ctx context.Context, consumer, method string) {
	event := &Event{Timestamp: time.Now().Unix(), Consumer: consumer, Method: method}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		event.Host = p.Addr.String()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for sub := range s.logs {
		select {
		case sub.events <- event:
		default:
			// Disconnect a slow subscriber instead of blocking unrelated RPCs.
			delete(s.logs, sub)
			close(sub.overflow)
		}
	}
	for stat := range s.stats {
		stat.ByMethod[method]++
		stat.ByConsumer[consumer]++
	}
}

func (s *microservice) unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	consumer, err := s.authorize(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}
	s.record(ctx, consumer, info.FullMethod)
	return handler(ctx, req)
}

func (s *microservice) stream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	consumer, err := s.authorize(stream.Context(), info.FullMethod)
	if err != nil {
		return err
	}
	s.record(stream.Context(), consumer, info.FullMethod)
	return handler(srv, stream)
}

func (s *microservice) Check(context.Context, *Nothing) (*Nothing, error) { return &Nothing{}, nil }
func (s *microservice) Add(context.Context, *Nothing) (*Nothing, error)   { return &Nothing{}, nil }
func (s *microservice) Test(context.Context, *Nothing) (*Nothing, error)  { return &Nothing{}, nil }

func (s *microservice) Logging(_ *Nothing, stream Admin_LoggingServer) error {
	sub := &logSubscription{events: make(chan *Event, 256), overflow: make(chan struct{})}
	s.mu.Lock()
	s.logs[sub] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.logs, sub); s.mu.Unlock() }()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-sub.overflow:
			return status.Error(codes.ResourceExhausted, "logging subscriber is too slow")
		case event := <-sub.events:
			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

func (s *microservice) Statistics(interval *StatInterval, stream Admin_StatisticsServer) error {
	// Check before conversion to avoid zero/overflow durations and ticker panics.
	if interval.IntervalSeconds == 0 || interval.IntervalSeconds > uint64((1<<63-1)/int64(time.Second)) {
		return status.Error(codes.InvalidArgument, "invalid statistics interval")
	}
	stat := &Stat{ByMethod: make(map[string]uint64), ByConsumer: make(map[string]uint64)}
	s.mu.Lock()
	s.stats[stat] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.stats, stat); s.mu.Unlock() }()
	ticker := time.NewTicker(time.Duration(interval.IntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			s.mu.Lock()
			snapshot := &Stat{Timestamp: time.Now().Unix(), ByMethod: stat.ByMethod, ByConsumer: stat.ByConsumer}
			stat.ByMethod = make(map[string]uint64)
			stat.ByConsumer = make(map[string]uint64)
			s.mu.Unlock()
			if err := stream.Send(snapshot); err != nil {
				return err
			}
		}
	}
}
