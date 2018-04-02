package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/akurin/golang-webservices/hw7_microservice/access"
	"github.com/akurin/golang-webservices/hw7_microservice/interceptors"
	"github.com/akurin/golang-webservices/hw7_microservice/logging"
	"github.com/akurin/golang-webservices/hw7_microservice/stat"
	"google.golang.org/grpc"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

func StartMyMicroservice(ctx context.Context, addr string, aclData string) error {
	emptyInterceptors := interceptors.Empty()

	collector := stat.NewCollector(emptyInterceptors)

	logger := logging.NewLogger(collector.Interceptors())

	acl, err := access.NewAcl(aclData, logger.Interceptors())
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	aclInterceptors := acl.Interceptors()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(aclInterceptors.Unary),
		grpc.StreamInterceptor(aclInterceptors.Stream),
	)

	RegisterAdminServer(server, adminServer{&logger, collector})
	RegisterBizServer(server, bizServer{})

	go func() {
		err := server.Serve(listener)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		<-ctx.Done()
		server.Stop()
		listener.Close()
	}()

	return nil
}

type adminServer struct {
	l             *logging.Logger
	statCollector *stat.Collector
}

func (s adminServer) Logging(n *Nothing, ls Admin_LoggingServer) error {
	subscription := s.l.Subscribe()
	for event := range subscription.Events {
		err := ls.Send(&Event{
			Timestamp: event.Timestamp.Unix(),
			Method:    event.Method,
			Consumer:  event.Consumer,
			Host:      event.Host,
		})
		// todo: delete chan
		if err != nil {
			subscription.Dispose()
			log.Println(err)
		}
	}
	return nil
}

func (s adminServer) Statistics(i *StatInterval, ss Admin_StatisticsServer) error {
	subscription := s.statCollector.Subscribe(time.Duration(i.IntervalSeconds) * time.Second)
	for event := range subscription.Events {
		err := ss.Send(&Stat{
			Timestamp:  event.Timestamp.Unix(),
			ByMethod:   event.ByMethod,
			ByConsumer: event.ByConsumer,
		})
		if err != nil {
			subscription.Dispose()
			log.Println(err)
		}
	}
	return nil
}

type bizServer struct{}

func (s bizServer) Check(ctx context.Context, n *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s bizServer) Add(ctx context.Context, n *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s bizServer) Test(ctx context.Context, n *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
