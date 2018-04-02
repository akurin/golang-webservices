package logging

import (
	"sync"
	"time"

	"github.com/akurin/golang-webservices/hw7_microservice/interceptors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Event struct {
	Timestamp time.Time
	Consumer  string
	Method    string
	Host      string
}

type Subscription struct {
	Events chan Event
}

func (s Subscription) Dispose() {
	close(s.Events)
}

type Logger struct {
	nextInterceptors interceptors.ServerInterceptors
	mutex            *sync.RWMutex
	subscriptions    []Subscription
}

func NewLogger(next interceptors.ServerInterceptors) Logger {
	return Logger{
		nextInterceptors: next,
		mutex:            &sync.RWMutex{},
	}
}

func (l *Logger) Subscribe() Subscription {
	subscription := Subscription{
		Events: make(chan Event),
	}

	l.mutex.Lock()
	l.subscriptions = append(l.subscriptions, subscription)
	l.mutex.Unlock()

	return subscription
}

func (l *Logger) Interceptors() interceptors.ServerInterceptors {
	return interceptors.ServerInterceptors{
		Unary:  l.interceptUnary,
		Stream: l.interceptStream,
	}
}

func (l *Logger) interceptUnary(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if err := l.log(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	reply, err := l.nextInterceptors.Unary(ctx, req, info, handler)
	return reply, err
}

func (l *Logger) log(ctx context.Context, fullMethod string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Internal, "cannot get metadata from incoming context")
	}
	consumers := md["consumer"]
	if len(consumers) == 0 {
		return status.Error(codes.Unauthenticated, "cannot get consumer")
	}
	p, ok := peer.FromContext(ctx)
	if !ok {
		return status.Error(codes.Internal, "cannot get peer from incoming context")
	}

	event := Event{
		Timestamp: time.Now(),
		Consumer:  consumers[0],
		Method:    fullMethod,
		Host:      p.Addr.String(),
	}

	var subscriptions []Subscription
	l.mutex.RLock()
	subscriptions = l.subscriptions
	l.mutex.RUnlock()

	for _, subscription := range subscriptions {
		subscription.Events <- event
	}
	return nil
}

func (l *Logger) interceptStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if err := l.log(ss.Context(), info.FullMethod); err != nil {
		return err
	}
	return l.nextInterceptors.Stream(srv, ss, info, handler)
}
