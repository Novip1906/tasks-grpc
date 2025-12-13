package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	pb "github.com/Novip1906/tasks-grpc/tasks/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/tasks/internal/config"
	"github.com/Novip1906/tasks-grpc/tasks/internal/contextkeys"
	"github.com/Novip1906/tasks-grpc/tasks/internal/elasticsearch"
	"github.com/Novip1906/tasks-grpc/tasks/internal/models"
	"github.com/Novip1906/tasks-grpc/tasks/internal/storage"
	"github.com/Novip1906/tasks-grpc/tasks/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TasksStorage interface {
	CreateTask(userId int64, text string) (id int64, err error)
	GetTaskById(userId int64, taskId int64) (*models.Task, error)
	GetAllUserTasks(userId int64) ([]*models.Task, error)
	UpdateTask(userId, taskId int64, newText string) (oldTask *models.Task, err error)
	DeleteTask(userId, taskId int64) (deletedTask *models.Task, err error)
}

type EmailSender interface {
	SendEventEmail(ctx context.Context, message *models.EventMessage) error
}

type TasksService struct {
	pb.UnimplementedTasksServiceServer
	cfg         *config.Config
	log         *slog.Logger
	db          TasksStorage
	emailSender EmailSender
	es          *elasticsearch.Client
}

func NewTasksService(config *config.Config, log *slog.Logger, db TasksStorage, emailSender EmailSender, esClient *elasticsearch.Client) *TasksService {
	return &TasksService{cfg: config, log: log, db: db, emailSender: emailSender, es: esClient}
}

func (s *TasksService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	tokenClaims, ok := contextkeys.GetTokenClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}
	log := contextkeys.GetLogger(ctx)

	text := req.GetText()

	log.Debug("attempt")

	text = processText(text)

	if !textIsValid(text, s.cfg) {
		log.Error("text len invalid")
		return nil, status.Error(codes.InvalidArgument, ErrInvalidTextMessage)
	}

	taskId, err := s.db.CreateTask(tokenClaims.UserId, text)
	if err != nil {
		log.Error("db error", logging.DbErr("CreateTask", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Info("task created")

	task := pb.Task{
		Id:         taskId,
		Text:       text,
		AuthorName: tokenClaims.Username,
		CreatedAt:  time.Now().Unix(),
	}

	if err := s.es.IndexTask(ctx, &models.Task{
		Id:         task.Id,
		Text:       task.Text,
		AuthorName: task.AuthorName,
		AuthorId:   tokenClaims.UserId,
		CreatedAt:  time.Now(),
	}); err != nil {
		log.Error("es index error", logging.Err(err))
	}

	if tokenClaims.Email == "" {
		return &pb.CreateTaskResponse{Task: &task}, nil
	}

	asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.emailSender.SendEventEmail(asyncCtx, &models.EventMessage{
		Email:    tokenClaims.Email,
		Username: tokenClaims.Username,
		Type:     "create",
		TaskText: text,
	})

	if err != nil {
		log.Error("kafka error", "email", tokenClaims.Email, logging.Err(err))
	} else {
		log.Info("kafka event message sent", "email", tokenClaims.Email, "event-type", "create")
	}

	return &pb.CreateTaskResponse{Task: &task}, nil

}

func (s *TasksService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	tokenClaims, ok := contextkeys.GetTokenClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}
	taskId := req.GetTaskId()

	log := contextkeys.GetLogger(ctx).With(slog.Int64("task_id", taskId))

	log.Debug("attempt")

	task, err := s.db.GetTaskById(tokenClaims.UserId, taskId)
	if errors.Is(err, storage.ErrTaskNotFound) {
		log.Error("task not found", logging.Err(err))
		return nil, status.Error(codes.NotFound, ErrTaskNotFoundMessage)
	}
	if errors.Is(err, storage.ErrNotTaskAuthor) {
		log.Error("user is not task author", logging.Err(err))
		return nil, status.Error(codes.PermissionDenied, ErrNotTaskAuthorMessage)
	}
	if err != nil {
		log.Error("db error", logging.DbErr("GetTask", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	return &pb.Task{
		Id:         task.Id,
		Text:       task.Text,
		AuthorName: task.AuthorName,
		CreatedAt:  task.CreatedAt.Unix(),
	}, nil

}

func (s *TasksService) SearchTask(ctx context.Context, req *pb.SearchTasksRequest) (*pb.SearchTasksResponse, error) {
	tokenClaims, ok := contextkeys.GetTokenClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}
	log := contextkeys.GetLogger(ctx)

	query := req.GetQuery()

	log.Debug("search tasks attempt", "query", query)

	tasks, err := s.es.Search(ctx, tokenClaims.UserId, query)
	if err != nil {
		log.Error("es search error", logging.Err(err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	pbTasks := make([]*pb.Task, 0, len(tasks))
	for _, t := range tasks {
		pbTasks = append(pbTasks, &pb.Task{
			Id:         t.Id,
			Text:       t.Text,
			AuthorName: t.AuthorName,
			CreatedAt:  t.CreatedAt.Unix(),
		})
	}

	return &pb.SearchTasksResponse{Tasks: pbTasks}, nil
}

func (s *TasksService) GetAllTasks(ctx context.Context, req *pb.GetAllTasksRequest) (*pb.GetAllTasksResponse, error) {
	tokenClaims, ok := contextkeys.GetTokenClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}
	log := contextkeys.GetLogger(ctx)

	log.Debug("get all tasks attempt")

	tasks, err := s.db.GetAllUserTasks(tokenClaims.UserId)
	if err != nil {
		log.Error("db error", logging.DbErr("GetAllTasks", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	pbTasks := make([]*pb.Task, 0, len(tasks))
	for _, task := range tasks {
		pbTask := &pb.Task{
			Id:         task.Id,
			Text:       task.Text,
			AuthorName: task.AuthorName,
			CreatedAt:  task.CreatedAt.Unix(),
		}
		pbTasks = append(pbTasks, pbTask)
	}

	return &pb.GetAllTasksResponse{Tasks: pbTasks}, nil

}

func (s *TasksService) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	tokenClaims, ok := contextkeys.GetTokenClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}
	taskId := req.GetTaskId()
	newText := req.GetNewText()

	log := contextkeys.GetLogger(ctx).With(slog.Int64("task_id", taskId))

	log.Debug("attempt")

	newText = processText(newText)

	if !textIsValid(newText, s.cfg) {
		log.Error("text len invalid")
		return nil, status.Error(codes.InvalidArgument, ErrInvalidTextMessage)
	}

	oldTask, err := s.db.UpdateTask(tokenClaims.UserId, taskId, newText)
	if errors.Is(err, storage.ErrTaskNotFound) {
		log.Error("task not found", logging.Err(err))
		return nil, status.Error(codes.NotFound, ErrTaskNotFoundMessage)
	}
	if errors.Is(err, storage.ErrNotTaskAuthor) {
		log.Error("user is not task author", logging.Err(err))
		return nil, status.Error(codes.PermissionDenied, ErrNotTaskAuthorMessage)
	}
	if err != nil {
		log.Error("db error", logging.DbErr("UpdateTask", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Info("task updated")

	if err := s.es.IndexTask(ctx, oldTask); err != nil {
		log.Error("es index error", logging.Err(err))
	}

	if tokenClaims.Email == "" {
		return &pb.UpdateTaskResponse{}, nil
	}

	asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.emailSender.SendEventEmail(asyncCtx, &models.EventMessage{
		Email:       tokenClaims.Email,
		Username:    tokenClaims.Username,
		Type:        "update",
		TaskText:    newText,
		TaskOldText: oldTask.Text,
	})

	if err != nil {
		log.Error("kafka error", "email", tokenClaims.Email, logging.Err(err))
	} else {
		log.Info("kafka event message sent", "email", tokenClaims.Email, "event-type", "update")
	}

	return &pb.UpdateTaskResponse{}, nil
}

func (s *TasksService) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	tokenClaims, ok := contextkeys.GetTokenClaims(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}
	taskId := req.GetTaskId()

	log := contextkeys.GetLogger(ctx).With(slog.Int64("task_id", taskId))

	log.Debug("attempt")

	task, err := s.db.DeleteTask(tokenClaims.UserId, taskId)
	if errors.Is(err, storage.ErrTaskNotFound) {
		log.Error("task not found", logging.Err(err))
		return nil, status.Error(codes.NotFound, ErrTaskNotFoundMessage)
	}
	if errors.Is(err, storage.ErrNotTaskAuthor) {
		log.Error("user is not task author", logging.Err(err))
		return nil, status.Error(codes.PermissionDenied, ErrNotTaskAuthorMessage)
	}
	if err != nil {
		log.Error("db error", logging.DbErr("DeleteTask", err))
		return nil, status.Error(codes.Internal, ErrInternalMessage)
	}

	log.Info("task deleted")

	responseTask := pb.Task{
		Id:         task.Id,
		Text:       task.Text,
		AuthorName: task.AuthorName,
		CreatedAt:  task.CreatedAt.Unix(),
	}

	if err := s.es.DeleteTask(ctx, task.Id); err != nil {
		log.Error("es delete error", logging.Err(err))
	}

	if tokenClaims.Email != "" {
		asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = s.emailSender.SendEventEmail(asyncCtx, &models.EventMessage{
			Email:    tokenClaims.Email,
			Username: tokenClaims.Username,
			Type:     "delete",
			TaskText: task.Text,
		})

		if err != nil {
			log.Error("kafka error", "email", tokenClaims.Email, logging.Err(err))
		} else {
			log.Info("kafka event message sent", "email", tokenClaims.Email, "event-type", "delete")
		}
	}

	return &pb.DeleteTaskResponse{Task: &responseTask}, nil
}

func textIsValid(text string, cfg *config.Config) bool {
	return len(text) >= cfg.Params.Text.Min && len(text) <= cfg.Params.Text.Max
}

func processText(text string) string {
	return strings.TrimSpace(text)
}
