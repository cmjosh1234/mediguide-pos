package aiworkergrpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mediguide/internal/aiworkerpb"
	"mediguide/internal/requestctx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Config struct {
	Address     string
	Secret      string
	CallTimeout time.Duration
	Retries     int
}

type Client struct {
	conn    *grpc.ClientConn
	client  aiworkerpb.AIWorkerServiceClient
	health  healthpb.HealthClient
	secret  string
	timeout time.Duration
	retries int
}

type RPCError struct {
	Code      codes.Code
	Message   string
	Retryable bool
}

func (e *RPCError) Error() string { return e.Message }

func NewClient(ctx context.Context, addr, secret string) (*Client, error) {
	return NewClientWithConfig(ctx, Config{
		Address: addr, Secret: secret,
		CallTimeout: 120 * time.Second, Retries: 1,
	})
}

func NewClientWithConfig(ctx context.Context, cfg Config) (*Client, error) {
	addr := strings.TrimSpace(cfg.Address)
	if addr == "" {
		return nil, fmt.Errorf("AI worker gRPC address is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = 120 * time.Second
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:    conn,
		client: aiworkerpb.NewAIWorkerServiceClient(conn),
		health:  healthpb.NewHealthClient(conn),
		secret:  strings.TrimSpace(cfg.Secret),
		timeout: cfg.CallTimeout,
		retries: cfg.Retries,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// AskRAG is safe to retry because it is stateless at the worker boundary.
func (c *Client) AskRAG(ctx context.Context, req *aiworkerpb.AskRAGRequest) (*aiworkerpb.AskRAGResponse, error) {
	var response *aiworkerpb.AskRAGResponse
	err := c.withRetry(ctx, true, func(callCtx context.Context) error {
		var err error
		response, err = c.client.AskRAG(callCtx, req)
		return err
	})
	return response, err
}

func (c *Client) Health(ctx context.Context) error {
	callCtx, cancel := c.deadlineContext(c.callContext(ctx))
	defer cancel()
	response, err := c.health.Check(callCtx, &healthpb.HealthCheckRequest{Service: "mediguide.aiworker.v1.AIWorkerService"})
	if err != nil {
		return normalizeError(err)
	}
	if response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return &RPCError{Code: codes.Unavailable, Message: "AI worker is not serving", Retryable: true}
	}
	return nil
}

func (c *Client) callContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if id := requestctx.CorrelationID(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-correlation-id", id)
	}
	if c.secret != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-worker-secret", c.secret)
	}
	return ctx
}

func (c *Client) deadlineContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= c.timeout {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) withRetry(ctx context.Context, safe bool, call func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	attempts := 1
	if safe {
		attempts += c.retries
	}
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		callCtx, cancel := c.deadlineContext(c.callContext(ctx))
		err := call(callCtx)
		cancel()
		if err == nil {
			return nil
		}
		last = normalizeError(err)
		var rpcErr *RPCError
		if !errors.As(last, &rpcErr) || !rpcErr.Retryable || attempt == attempts-1 {
			return last
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 150 * time.Millisecond):
		}
	}
	return last
}

func normalizeError(err error) error {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	if !ok {
		return err
	}
	retryable := s.Code() == codes.Unavailable || s.Code() == codes.ResourceExhausted
	return &RPCError{Code: s.Code(), Message: s.Message(), Retryable: retryable}
}
