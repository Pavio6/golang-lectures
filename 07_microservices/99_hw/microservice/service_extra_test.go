package main

import (
	"context"
	"testing"

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
