package server

import (
	"context"
	"io"
	"log"
	"slices"
	"time"

	pb "github.com/araminian/grpc-simple-app/proto/todo/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (s *Server) AddTask(
	_ context.Context,
	in *pb.AddTaskRequest,
) (*pb.AddTaskResponse, error) {

	if err := in.Validate(); err != nil {
		return nil, err
	}

	id, err := s.D.addTask(in.Description, in.DueDate.AsTime())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add task: %v", err)
	}

	return &pb.AddTaskResponse{Id: id}, nil
}

func (s *Server) ListTasks(
	req *pb.ListTasksRequest,
	stream pb.TodoService_ListTasksServer,
) error {
	return s.D.getTasks(func(i interface{}) error {
		task := i.(*pb.Task)

		// filter
		Filter(task, req.Mask)

		overdue := task.DueDate != nil && !task.Done && task.DueDate.AsTime().Before(time.Now().UTC())

		log.Printf("Server: sending task: %v", task)
		log.Printf("Server: overdue: %v", overdue)

		err := stream.Send(&pb.ListTasksResponse{
			Task:    task,
			Overdue: overdue,
		})
		return err
	})
}

func (s *Server) UpdateTask(stream pb.TodoService_UpdateTaskServer) error {

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.UpdateTaskResponse{})
		}
		if err != nil {
			return err
		}

		s.D.updateTask(req.Id, req.Description, req.DueDate.AsTime(), req.Done)
	}
}

func (s *Server) DeleteTask(stream pb.TodoService_DeleteTaskServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.D.deleteTask(req.Id)
		stream.Send(&pb.DeleteTaskResponse{})
	}
}

func Filter(msg proto.Message, mask *fieldmaskpb.FieldMask) {

	if mask == nil || len(mask.Paths) == 0 {
		return
	}

	rft := msg.ProtoReflect()

	rft.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if !slices.Contains(mask.Paths, string(fd.Name())) {
			rft.Clear(fd)
		}
		return true
	})

}
