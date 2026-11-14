package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestACLServiceBoundary(t *testing.T) {
	s := &microservice{acl: map[string][]string{"admin": {"/main.Biz/*"}}}
	for _, tc := range []struct {
		name      string
		consumers []string
		method    string
		allowed   bool
	}{
		{"wildcard", []string{"admin"}, "/main.Biz/Test", true},
		{"other service", []string{"admin"}, "/main.BizExtra/Test", false},
		{"missing consumer", nil, "/main.Biz/Test", false},
		{"ambiguous consumer", []string{"admin", "unknown"}, "/main.Biz/Test", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{"consumer": tc.consumers})
			_, err := s.authorize(ctx, tc.method)
			if tc.allowed && err != nil {
				t.Fatal(err)
			}
			if !tc.allowed && status.Code(err) != codes.Unauthenticated {
				t.Fatalf("expected Unauthenticated, got %v", err)
			}
		})
	}
}

func TestStatisticsInvalidInterval(t *testing.T) {
	s := &microservice{}
	for _, interval := range []uint64{0, 1<<64 - 1} {
		if err := s.Statistics(&StatInterval{IntervalSeconds: interval}, nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("interval %d: expected InvalidArgument, got %v", interval, err)
		}
	}
}

// A stalled reader must not lose its subscription or block unrelated calls.
type delayedLogStream struct {
	grpc.ServerStream
	ctx      context.Context
	release  chan struct{}
	received chan *Event
}

func (s *delayedLogStream) Context() context.Context { return s.ctx }
func (s *delayedLogStream) Send(event *Event) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.release:
	}
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case s.received <- event:
		return nil
	}
}

func TestLoggingBacklogAndCancellation(t *testing.T) {
	s := &microservice{logs: make(map[*logSubscription]struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream := &delayedLogStream{ctx: ctx, release: make(chan struct{}), received: make(chan *Event)}
	done := make(chan error, 1)
	go func() { done <- s.Logging(&Nothing{}, stream) }()
	for {
		s.mu.Lock()
		ready := len(s.logs) == 1
		s.mu.Unlock()
		if ready {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("subscription not registered")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	for i := 0; i < 1024; i++ {
		s.record(ctx, "user", fmt.Sprint(i))
	}
	close(stream.release)
	for i := 0; i < 1024; i++ {
		select {
		case event := <-stream.received:
			if event.Method != fmt.Sprint(i) {
				t.Fatalf("event %d: got %s", i, event.Method)
			}
		case <-ctx.Done():
			t.Fatal("backlog was not delivered")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("logging did not exit")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.logs) != 0 {
		t.Fatal("subscription was not removed")
	}
}
