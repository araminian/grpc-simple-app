package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/araminian/grpc-simple-app/proto/todo/v2"
)

func AddTask(c pb.TodoServiceClient, description string, dueDate time.Time, headers map[string]string) uint64 {
	req := &pb.AddTaskRequest{
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}

	// Add headers
	md := metadata.New(headers)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	res, err := c.AddTask(ctx, req, grpc.UseCompressor(gzip.Name)) // Enable gzip compression for this call
	if err != nil {
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.InvalidArgument, codes.Internal:
				log.Fatalf("Client: Code: %s, Message: %s", s.Code(), s.Message())
			default:
				log.Fatal(s)
			}
		} else {
			panic(err)
		}
	}
	log.Printf("Client: added task with id: %d", res.Id)
	return res.Id
}

func NewClient(addr string) (pb.TodoServiceClient, *grpc.ClientConn, error) {

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Disable TLS
		//grpc.WithTransportCredentials(creds), // Enable TLS
		grpc.WithUnaryInterceptor(unaryAuthInterceptor),
		grpc.WithStreamInterceptor(streamAuthInterceptor),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)), // Enable gzip compression
	}
	conn, err := grpc.Dial(addr, opts...)

	if err != nil {
		return nil, nil, err
	}

	return pb.NewTodoServiceClient(conn), conn, nil

}

func NewMask() (*fieldmaskpb.FieldMask, error) {
	mask, err := fieldmaskpb.New(&pb.Task{}, "id", "description", "due_date", "done")
	if err != nil {
		return nil, err
	}
	return mask, nil
}

func PrintTasks(c pb.TodoServiceClient, mask *fieldmaskpb.FieldMask, headers map[string]string) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add headers
	md := metadata.New(headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &pb.ListTasksRequest{Mask: mask}

	stream, err := c.ListTasks(ctx, req)
	if err != nil {
		log.Fatalf("error listing tasks: %v", err)
	}

	for {
		res, err := stream.Recv()

		//Temp , should be removed in next section

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving tasks: %v", err)
		}
		fmt.Println(res.Task.String(), "overdue:", res.Overdue)
	}
}

func UpdateTask(
	c pb.TodoServiceClient,
	reqs []*pb.UpdateTaskRequest,
	headers map[string]string,
) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add headers
	md := metadata.New(headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := c.UpdateTask(ctx)

	if err != nil {
		log.Fatalf("error updating task: %v", err)
	}

	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			log.Fatalf("error sending task: %v", err)
		}
		if req.Id != 0 {
			log.Printf("Client: updated task with id: %d", req.Id)
		}
	}

	if _, err := stream.CloseAndRecv(); err != nil {
		log.Fatalf("error closing stream: %v", err)
	}
}

func DeleteTask(c pb.TodoServiceClient, headers map[string]string, reqs ...*pb.DeleteTaskRequest) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add headers
	md := metadata.New(headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := c.DeleteTask(ctx)

	if err != nil {
		log.Fatalf("error deleting task: %v", err)
	}

	// wait for response
	waitc := make(chan struct{})

	// Handle responses
	go func() {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("error receiving task: %v", err)
			}

			log.Println("Client: deleted task")
		}
	}()

	// Send requests
	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			log.Fatalf("error sending task: %v", err)
		}
	}

	// Close stream
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("error closing stream: %v", err)
	}

	// Wait for response
	<-waitc
}
