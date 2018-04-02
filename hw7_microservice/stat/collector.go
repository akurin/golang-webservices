package stat

import (
	"context"
	"sync"
	"time"

	"github.com/akurin/golang-webservices/hw7_microservice/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Collector struct {
	nextInterceptors interceptors.ServerInterceptors
	mutex            *sync.Mutex
	subscriptions    []Subscription
}

func NewCollector(nextInterceptors interceptors.ServerInterceptors) *Collector {
	return &Collector{
		nextInterceptors: nextInterceptors,
		mutex:            &sync.Mutex{},
	}
}

func (c *Collector) Subscribe(interval time.Duration) *Subscription {
	ticker := time.NewTicker(interval)

	subscription := &Subscription{
		calls:  make(chan call),
		Events: make(chan Event),
		ticker: ticker,
	}

	c.mutex.Lock()
	c.subscriptions = append(c.subscriptions, *subscription)
	c.mutex.Unlock()

	byConsumer := make(map[string]uint64)
	byMethod := make(map[string]uint64)

	go func() {
		for {
			select {
			case message := <-subscription.calls:
				byMethod[message.method] = byMethod[message.method] + 1
				byConsumer[message.consumer] = byConsumer[message.consumer] + 1
			case _ = <-ticker.C:
				subscription.Events <- Event{
					Timestamp:  time.Now(),
					ByConsumer: byConsumer,
					ByMethod:   byMethod,
				}
				byConsumer = make(map[string]uint64)
				byMethod = make(map[string]uint64)
			}
		}
	}()
	return subscription
}

func (c *Collector) Interceptors() interceptors.ServerInterceptors {
	return interceptors.ServerInterceptors{
		Unary:  c.interceptUnary,
		Stream: c.interceptStream,
	}
}

func (c *Collector) interceptUnary(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if err := c.updateStat(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	reply, err := c.nextInterceptors.Unary(ctx, req, info, handler)
	return reply, err
}

func (c *Collector) updateStat(ctx context.Context, method string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Internal, "cannot get metadata from incoming context")
	}
	consumers := md["consumer"]
	if len(consumers) == 0 {
		return status.Error(codes.Unauthenticated, "cannot get consumer")
	}
	consumer := consumers[0]

	c.mutex.Lock()
	for _, subscr := range c.subscriptions {
		subscr.calls <- call{
			method:   method,
			consumer: consumer,
		}
	}
	c.mutex.Unlock()
	return nil
}

func (c *Collector) interceptStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if err := c.updateStat(ss.Context(), info.FullMethod); err != nil {
		return err
	}
	return handler(srv, ss)
}
