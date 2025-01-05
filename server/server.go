package server

import (
	"log"
	"os"

	pb "github.com/araminian/grpc-simple-app/proto/todo/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

type Server struct {
	D db
	pb.UnimplementedTodoServiceServer
}

func NewServer() (*grpc.Server, error) {

	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)

	var opts []grpc.ServerOption = []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			auth.UnaryServerInterceptor(validateAuthToken),
			logging.UnaryServerInterceptor(logCalls(logger)),
		),
		grpc.ChainStreamInterceptor(
			auth.StreamServerInterceptor(validateAuthToken),
			logging.StreamServerInterceptor(logCalls(logger)),
		),
		//grpc.Creds(creds), // Add TLS credentials
	}
	s := grpc.NewServer(opts...)

	pb.RegisterTodoServiceServer(s, &Server{D: New()})

	return s, nil
}
