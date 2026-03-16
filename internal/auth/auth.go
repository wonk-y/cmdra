package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type contextKey string

const identityKey contextKey = "client-cn"

// IdentityFromContext returns the authenticated client CN attached by the interceptors.
func IdentityFromContext(ctx context.Context) (string, bool) {
	cn, ok := ctx.Value(identityKey).(string)
	return cn, ok
}

// UnaryAuthInterceptor enforces mTLS authentication and an optional CN allowlist.
func UnaryAuthInterceptor(allowed map[string]struct{}) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		nextCtx, err := authenticate(ctx, allowed)
		if err != nil {
			return nil, err
		}
		return handler(nextCtx, req)
	}
}

// StreamAuthInterceptor is the streaming equivalent of UnaryAuthInterceptor.
func StreamAuthInterceptor(allowed map[string]struct{}) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		nextCtx, err := authenticate(stream.Context(), allowed)
		if err != nil {
			return err
		}
		return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: nextCtx})
	}
}

// ParseAllowList splits a comma-separated CN list into a lookup set.
func ParseAllowList(raw string) map[string]struct{} {
	allowed := map[string]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		allowed[item] = struct{}{}
	}
	return allowed
}

func authenticate(ctx context.Context, allowed map[string]struct{}) (context.Context, error) {
	cn, err := extractCN(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "mtls authentication failed: %v", err)
	}
	if len(allowed) > 0 {
		if _, ok := allowed[cn]; !ok {
			return nil, status.Error(codes.PermissionDenied, "client CN not authorized")
		}
	}
	return context.WithValue(ctx, identityKey, cn), nil
}

func extractCN(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", errors.New("missing peer context")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", errors.New("peer auth is not TLS")
	}
	cn := verifiedChainCN(tlsInfo.State)
	if cn == "" {
		return "", errors.New("client certificate CN is empty")
	}
	return cn, nil
}

func verifiedChainCN(state tls.ConnectionState) string {
	if len(state.VerifiedChains) > 0 && len(state.VerifiedChains[0]) > 0 {
		return strings.TrimSpace(state.VerifiedChains[0][0].Subject.CommonName)
	}
	if len(state.PeerCertificates) > 0 {
		return strings.TrimSpace(state.PeerCertificates[0].Subject.CommonName)
	}
	return ""
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context overrides the wrapped stream context with the authenticated context.
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
