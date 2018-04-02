package interceptors

import (
	"google.golang.org/grpc"
)

type ServerInterceptors struct {
	Unary  grpc.UnaryServerInterceptor
	Stream grpc.StreamServerInterceptor
}
