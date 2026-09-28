package aiworkergrpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeErrorClassifiesTransientCodes(t *testing.T) {
	err := normalizeError(status.Error(codes.Unavailable, "worker unavailable"))
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected RPCError, got %T", err)
	}
	if rpcErr.Code != codes.Unavailable || !rpcErr.Retryable {
		t.Fatalf("unexpected classification: %#v", rpcErr)
	}

	err = normalizeError(status.Error(codes.InvalidArgument, "bad request"))
	if !errors.As(err, &rpcErr) || rpcErr.Retryable {
		t.Fatalf("invalid argument must not be retryable: %#v", err)
	}
}

func TestDeadlineContextUsesShorterClientDeadline(t *testing.T) {
	client := &Client{timeout: 50 * time.Millisecond}
	parent, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ctx, done := client.deadlineContext(parent)
	defer done()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 200*time.Millisecond {
		t.Fatalf("expected client timeout to cap deadline, remaining=%s", remaining)
	}
}
