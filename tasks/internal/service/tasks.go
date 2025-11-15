package service

import (
	"context"
	"errors"
	"log/slog"

	pb "github.com/Novip1906/tasks-grpc/tasks/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/tasks/internal/config"
	"github.com/Novip1906/tasks-grpc/tasks/internal/models"
	"github.com/Novip1906/tasks-grpc/tasks/internal/storage"
	"github.com/Novip1906/tasks-grpc/tasks/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TasksStorage interface {
	CreateTask(userId int64, text string) error
	GetTask(userId int64, taskId int64) (*models.Task, error)
	GetAllTasks(userId int64) ([]*models.Task, error)
	UpdateTask(userId int64, taskId int64, newText string) error
	DeleteTask(userId int64, taskId int64) error
}

type TasksService struct {
	pb.UnimplementedTasksServiceServer
	cfg *config.Config
	log *slog.Logger
	db  TasksStorage
}

func NewTasksService(config *config.Config, log *slog.Logger, db TasksStorage) *TasksService {
	return &TasksService{cfg: config, log: log, db: db}
}

func (s *TasksService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	userId, ok := ctx.Value(models.UserIDContextKey).(int64)
	if !ok {
		return nil, status.Error(codes.Internal, "!ok")
	}

	text := req.GetText()

	log := s.log.With(
		slog.String("op", "Tasks.CreateTask"),
		slog.Int64("user_id", userId),
		slog.String("text", text),
	)

	log.Info("attempt")

	if !textIsValid(text, s.cfg) {
		log.Error("text len invalid")
		return nil, status.Error(codes.InvalidArgument, "invalid text length")
	}

	err := s.db.CreateTask(userId, text)
	if err != nil {
		log.Error("db.CreateTask", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	log.Info("task created")

	return &pb.CreateTaskResponse{}, nil

}

func (s *TasksService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	userId := ctx.Value(models.UserIDContextKey).(int64)
	taskId := req.GetTaskId()

	log := s.log.With(
		slog.String("op", "Tasks.GetTask"),
		slog.Int64("user_id", userId),
		slog.Int64("task_id", taskId),
	)

	log.Info("attempt")

	task, err := s.db.GetTask(userId, taskId)
	if errors.Is(err, storage.ErrTaskNotFound) {
		log.Error(err.Error())
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, storage.ErrNotTaskAuthor) {
		log.Error(err.Error())
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if err != nil {
		log.Error("db.GetTask", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	log.Info("completed")

	return &pb.Task{Text: task.Text, AuthorName: task.AuthorName, CreatedAt: task.CreatedAt.Unix()}, nil

}

func (s *TasksService) GetAllTasks(ctx context.Context, req *pb.GetAllTasksRequest) (*pb.GetAllTasksResponse, error) {
	userId := ctx.Value(models.UserIDContextKey).(int64)

	log := s.log.With(
		slog.String("op", "Tasks.GetAllTasks"),
		slog.Int64("user_id", userId),
	)

	log.Info("get all tasks attempt")

	tasks, err := s.db.GetAllTasks(userId)
	if err != nil {
		log.Error("db.GetAllTasks", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	pbTasks := make([]*pb.Task, 0, len(tasks))
	for _, task := range tasks {
		pbTask := &pb.Task{
			Text:       task.Text,
			AuthorName: task.AuthorName,
			CreatedAt:  task.CreatedAt.Unix(),
		}
		pbTasks = append(pbTasks, pbTask)
	}

	log.Info("completed")

	return &pb.GetAllTasksResponse{Tasks: pbTasks}, nil

}

func (s *TasksService) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	userId := ctx.Value(models.UserIDContextKey).(int64)
	taskId := req.GetTaskId()
	newText := req.GetNewText()

	log := s.log.With(
		slog.String("op", "Tasks.UpdateTask"),
		slog.Int64("user_id", userId),
		slog.Int64("task_id", taskId),
		slog.String("new_text", newText),
	)

	log.Info("attempt")

	if !textIsValid(newText, s.cfg) {
		log.Error("text len invalid")
		return nil, status.Error(codes.InvalidArgument, "invalid text length")
	}

	err := s.db.UpdateTask(userId, taskId, newText)
	if errors.Is(err, storage.ErrTaskNotFound) {
		log.Error(err.Error())
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, storage.ErrNotTaskAuthor) {
		log.Error(err.Error())
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if err != nil {
		log.Error("db.UpdateTask", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	log.Info("task updated")

	return &pb.UpdateTaskResponse{}, nil
}

func (s *TasksService) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	userId := ctx.Value(models.UserIDContextKey).(int64)
	taskId := req.GetTaskId()

	log := s.log.With(
		slog.String("op", "Tasks.CreateTask"),
		slog.Int64("user_id", userId),
		slog.Int64("task_id", taskId),
	)

	log.Info("attempt")

	err := s.db.DeleteTask(userId, taskId)
	if errors.Is(err, storage.ErrTaskNotFound) {
		log.Error(err.Error())
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, storage.ErrNotTaskAuthor) {
		log.Error(err.Error())
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if err != nil {
		log.Error("db.DeleteTask", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternal.Error())
	}

	log.Info("task deleted")

	return &pb.DeleteTaskResponse{}, nil
}

func textIsValid(text string, cfg *config.Config) bool {
	return len(text) >= cfg.Params.Text.Min && len(text) <= cfg.Params.Text.Max
}
