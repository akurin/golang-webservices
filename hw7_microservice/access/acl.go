package access

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/akurin/golang-webservices/hw7_microservice/interceptors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/transport"
)

type parsedAclData map[string][]string

type Acl struct {
	data parsedAclData
	next interceptors.ServerInterceptors
}

func NewAcl(aclData string, next interceptors.ServerInterceptors) (Acl, error) {
	var parsedAclData parsedAclData
	if err := json.Unmarshal([]byte(aclData), &parsedAclData); err != nil {
		return Acl{}, err
	}
	return Acl{parsedAclData, next}, nil
}

func (l Acl) Interceptors() interceptors.ServerInterceptors {
	return interceptors.ServerInterceptors{
		Unary:  l.interceptUnary,
		Stream: l.interceptStream,
	}
}

func (l Acl) interceptUnary(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if err := l.data.checkAccess(ctx); err != nil {
		return nil, err
	}

	reply, err := l.next.Unary(ctx, req, info, handler)
	return reply, err
}

func (d parsedAclData) checkAccess(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Internal, "cannot get metadata from incoming context")
	}
	consumers := md["consumer"]
	if len(consumers) == 0 {
		return status.Error(codes.Unauthenticated, "cannot get consumer")
	}

	stream, ok := transport.StreamFromContext(ctx)
	if !ok {
		return status.Error(codes.Internal, "cannot get stream")
	}

	allowedMethods := d[consumers[0]]
	if !match(allowedMethods, stream.Method()) {
		return status.Error(codes.Unauthenticated, "access denied")
	}

	return nil
}

func match(patterns []string, s string) bool {
	for _, pattern := range patterns {
		matchExpr := strings.Replace(regexp.QuoteMeta(pattern), "\\*", "*", -1)
		regexpr := regexp.MustCompile(matchExpr)
		if regexpr.Match([]byte(s)) {
			return true
		}
	}
	return false
}

func (l Acl) interceptStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if err := l.data.checkAccess(ss.Context()); err != nil {
		return err
	}

	return l.next.Stream(srv, ss, info, handler)
}
