package server

import (
	"fmt"
	"time"

	pb "github.com/araminian/grpc-simple-app/proto/todo/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type inMemoryDb struct {
	tasks []*pb.Task
}

func New() db {
	return &inMemoryDb{}
}

func (d *inMemoryDb) addTask(description string, dueDate time.Time) (uint64, error) {
	nextId := uint64(len(d.tasks) + 1)
	task := &pb.Task{
		Id:          nextId,
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}

	d.tasks = append(d.tasks, task)

	return nextId, nil
}

func (d *inMemoryDb) getTasks(f func(interface{}) error) error {
	for _, task := range d.tasks {
		if err := f(task); err != nil {
			return err
		}
	}

	return nil
}

func (d *inMemoryDb) updateTask(id uint64, description string, dueDate time.Time, done bool) error {
	for _, task := range d.tasks {
		if task.Id == id {
			task.Description = description
			task.DueDate = timestamppb.New(dueDate)
			task.Done = done
			return nil
		}
	}

	return fmt.Errorf("task with id %d not found", id)
}

func (d *inMemoryDb) deleteTask(id uint64) error {
	for i, task := range d.tasks {
		if task.Id == id {
			d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("task with id %d not found", id)
}
