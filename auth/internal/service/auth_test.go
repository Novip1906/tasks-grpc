package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	pb "github.com/Novip1906/tasks-grpc/auth/api/proto/gen"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/internal/contextkeys"
	"github.com/Novip1906/tasks-grpc/auth/internal/models"
	"github.com/Novip1906/tasks-grpc/auth/internal/storage"
	"github.com/Novip1906/tasks-grpc/auth/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Mocks ---

type MockUserStorage struct {
	mock.Mock
}

func (m *MockUserStorage) CheckUsernamePassword(username, password string) (int64, string, error) {
	args := m.Called(username, password)
	return args.Get(0).(int64), args.String(1), args.Error(2)
}

func (m *MockUserStorage) CheckEmailExists(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserStorage) AddUser(username, password, email string) (int64, error) {
	args := m.Called(username, password, email)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserStorage) SetEmail(userId int64, email string) error {
	args := m.Called(userId, email)
	return args.Error(0)
}

type MockCodeStorage struct {
	mock.Mock
}

func (m *MockCodeStorage) SetCode(ctx context.Context, email, code string, userId int64) error {
	args := m.Called(ctx, email, code, userId)
	return args.Error(0)
}

func (m *MockCodeStorage) GetCode(ctx context.Context, email string) (string, int64, error) {
	args := m.Called(ctx, email)
	return args.String(0), args.Get(1).(int64), args.Error(2)
}

func (m *MockCodeStorage) DeleteCode(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendVerificationEmail(ctx context.Context, message *models.EmailVerificationMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// --- Setup Helper ---

func setupService() (*AuthService, *MockUserStorage, *MockCodeStorage, *MockEmailSender) {
	cfg := &config.Config{
		JWTSecretKey: "test-secret",
		Params: config.Params{
			Username: config.MinMaxLen{Min: 3, Max: 20},
			Password: config.MinMaxLen{Min: 6, Max: 20},
		},
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userDb := new(MockUserStorage)
	codeDb := new(MockCodeStorage)
	emailSender := new(MockEmailSender)

	service := NewAuthService(cfg, log, userDb, codeDb, emailSender)
	return service, userDb, codeDb, emailSender
}

func getCtx() context.Context {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return contextkeys.WithLogger(ctx, log)
}

// --- Tests ---

func TestLogin_Success(t *testing.T) {
	s, mockUser, _, _ := setupService()
	ctx := getCtx()

	mockUser.On("CheckUsernamePassword", "john", "secret123").Return(int64(1), "john@test.com", nil)

	req := &pb.LoginRequest{Username: "john", Password: "secret123"}
	resp, err := s.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	mockUser.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	s, mockUser, _, _ := setupService()
	ctx := getCtx()

	mockUser.On("CheckUsernamePassword", "unknown", "pass").Return(int64(0), "", storage.ErrUserNotFound)

	req := &pb.LoginRequest{Username: "unknown", Password: "pass"}
	resp, err := s.Login(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	mockUser.AssertExpectations(t)
}

func TestRegister_Success_WithEmail(t *testing.T) {
	s, mockUser, mockCode, mockEmail := setupService()
	ctx := getCtx()

	mockCode.On("GetCode", ctx, "john@test.com").Return("", int64(0), storage.ErrCodeNotFound)
	mockUser.On("CheckEmailExists", "john@test.com").Return(false, nil)
	mockUser.On("AddUser", "john", "secret123", "").Return(int64(1), nil)
	mockCode.On("SetCode", ctx, "john@test.com", mock.AnythingOfType("string"), int64(1)).Return(nil)
	mockEmail.On("SendVerificationEmail", mock.Anything, mock.AnythingOfType("*models.EmailVerificationMessage")).Return(nil)

	req := &pb.RegisterRequest{
		Username: "john",
		Email:    "john@test.com",
		Password: "secret123",
	}
	resp, err := s.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockUser.AssertExpectations(t)
	mockCode.AssertExpectations(t)
	mockEmail.AssertExpectations(t)
}

func TestRegister_InvalidInputs(t *testing.T) {
	s, _, _, _ := setupService()
	ctx := getCtx()

	req := &pb.RegisterRequest{
		Username: "ab", // Too short
		Email:    "invalid",
		Password: "pass", // Too short
	}
	resp, err := s.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestValidateToken_Success(t *testing.T) {
	s, _, _, _ := setupService()
	ctx := getCtx()

	token, _ := utils.EncodeJWTToken(1, "test@test.com", "test", "test-secret")

	req := &pb.ValidateTokenRequest{Token: token}
	resp, err := s.ValidateToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.UserId)
	assert.Equal(t, "test", resp.Username)
	assert.Equal(t, "test@test.com", resp.Email)
}

func TestValidateVerificationCode_Success(t *testing.T) {
	s, mockUser, mockCode, _ := setupService()
	ctx := getCtx()

	mockCode.On("GetCode", ctx, "test@test.com").Return("1234", int64(1), nil)
	mockUser.On("SetEmail", int64(1), "test@test.com").Return(nil)
	mockCode.On("DeleteCode", ctx, "test@test.com").Return(nil)

	req := &pb.ValidateCodeRequest{Email: "test@test.com", Code: "1234"}
	resp, err := s.ValidateVerificationCode(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockCode.AssertExpectations(t)
	mockUser.AssertExpectations(t)
}

func TestChangeEmail_Success(t *testing.T) {
	s, _, mockCode, mockEmail := setupService()
	ctx := getCtx()

	// Inject claims into context
	claims := &contextkeys.TokenClaims{
		UserId:   1,
		Email:    "old@test.com",
		Username: "john",
	}
	ctx = contextkeys.WithTokenClaims(ctx, claims)

	mockCode.On("GetCode", ctx, "new@test.com").Return("", int64(0), storage.ErrCodeNotFound)
	mockCode.On("SetCode", ctx, "new@test.com", mock.AnythingOfType("string"), int64(1)).Return(nil)
	mockEmail.On("SendVerificationEmail", mock.Anything, mock.AnythingOfType("*models.EmailVerificationMessage")).Return(nil)

	req := &pb.ChangeEmailRequest{NewEmail: "new@test.com"}
	resp, err := s.ChangeEmail(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockCode.AssertExpectations(t)
	mockEmail.AssertExpectations(t)
}
