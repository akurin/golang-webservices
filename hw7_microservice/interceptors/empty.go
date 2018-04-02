package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

func Empty() ServerInterceptors {
	empty := emptyInterceptor{}
	return ServerInterceptors{
		Unary:  empty.interceptUnary,
		Stream: empty.interceptStream,
	}
}

type emptyInterceptor struct{}

func (i emptyInterceptor) interceptUnary(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	reply, err := handler(ctx, req)
	return reply, err
}

func (i emptyInterceptor) interceptStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	return handler(srv, ss)
}
