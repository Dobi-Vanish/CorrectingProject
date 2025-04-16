package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"reward-service/api/calltypes"
	"reward-service/internal/service"
	"reward-service/pkg/errormsg"
	"strconv"
	"strings"
	"testing"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
)

// MockRepository - мок репозитория для тестирования.
type MockRepository struct {
	mock.Mock
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func (m *MockRepository) Insert(user calltypes.User) (int, error) {
	args := m.Called(user)

	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetAll() ([]*calltypes.User, error) {
	args := m.Called()

	users, ok := args.Get(0).([]*calltypes.User)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: expected []*calltypes.User, got %T", args.Get(0)) //nolint: err113
	}

	return users, args.Error(1) //nolint: wrapcheck
}

func (m *MockRepository) GetOne(id int) (*calltypes.User, error) {
	args := m.Called(id)

	user, ok := args.Get(0).(*calltypes.User)
	if !ok {
		return nil, fmt.Errorf("type assertion to *calltypes.User failed, got %T", args.Get(0)) //nolint: err113
	}

	return user, args.Error(1) //nolint: wrapcheck
}

func (m *MockRepository) GetByEmail(email string) (*calltypes.User, error) {
	args := m.Called(email)

	user, ok := args.Get(0).(*calltypes.User)
	if !ok {
		return nil, fmt.Errorf("type assertion to *calltypes.User failed, got %T", args.Get(0)) //nolint: err113
	}

	return user, args.Error(1) //nolint: wrapcheck
}

func (m *MockRepository) PasswordMatches(password string, user calltypes.User) (bool, error) {
	args := m.Called(password, user)

	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) AddPoints(id int, points int) error {
	args := m.Called(id, points)

	return args.Error(0) //nolint: wrapcheck
}

func (m *MockRepository) RedeemReferrer(id int, referrer string) error {
	args := m.Called(id, referrer)

	return args.Error(0) //nolint: wrapcheck
}

func (m *MockRepository) Update(user calltypes.User) error {
	args := m.Called(user)

	return args.Error(0) //nolint: wrapcheck
}

func (m *MockRepository) EmailCheck(email string) (*calltypes.User, error) {
	args := m.Called(email)

	user, ok := args.Get(0).(*calltypes.User)
	if !ok {
		return nil, fmt.Errorf("type assertion to *calltypes.User failed, got %T", args.Get(0)) //nolint: err113
	}

	return user, args.Error(1) //nolint: wrapcheck
}

func (m *MockRepository) UpdateScore(user calltypes.User) error {
	args := m.Called(user)

	return args.Error(0) //nolint: wrapcheck
}

func (m *MockRepository) StoreRefreshToken(userID int, hashedToken string) error {
	args := m.Called(userID, hashedToken)

	return args.Error(0) //nolint: wrapcheck
}

func TestRewardService_Registrate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    string
		mockSetup      func(*MockRepository)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Successful registration",
			requestBody: `{
				"email": "test@example.com",
				"firstName": "Test",
				"lastName": "User",
				"password": "securepassword123"
			}`,
			mockSetup: func(m *MockRepository) {
				m.On("Insert", mock.AnythingOfType("calltypes.User")).Return(1, nil)
			},
			expectedStatus: http.StatusAccepted,
			expectedError:  false,
		},
		{
			name: "Short password",
			requestBody: `{
				"email": "test@example.com",
				"firstName": "Test",
				"lastName": "User",
				"password": "short"
			}`,
			mockSetup:      func(_ *MockRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "Repository error",
			requestBody: `{
				"email": "test@example.com",
				"firstName": "Test",
				"lastName": "User",
				"password": "securepassword123"
			}`,
			mockSetup: func(m *MockRepository) {
				m.On("Insert", mock.AnythingOfType("calltypes.User")).Return(0, errormsg.ErrRepositoryError)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := service.NewRewardService(mockRepo)

			req := httptest.NewRequest(http.MethodPost, "/registrate", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			svc.Registrate(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.mockSetup != nil {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestRewardService_Authenticate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    string
		mockSetup      func(*MockRepository)
		expectedStatus int
	}{
		{
			name: "Successful authentication",
			requestBody: `{
                "email": "test@example.com",
                "password": "correctpassword"
            }`,
			mockSetup: func(m *MockRepository) { //nolint:varnamelen
				user := &calltypes.User{
					ID:        1,
					Email:     "test@example.com",
					FirstName: "Test",
					LastName:  "User",
					Password:  "hashedpassword",
				}
				m.On("GetByEmail", "test@example.com").Return(user, nil)
				m.On("PasswordMatches", "correctpassword", *user).Return(true, nil)
				m.On("StoreRefreshToken", user.ID, mock.AnythingOfType("string")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid credentials",
			requestBody: `{
                "email": "test@example.com",
                "password": "wrongpassword"
            }`,
			mockSetup: func(m *MockRepository) { //nolint:varnamelen
				user := &calltypes.User{
					ID:        1,
					Email:     "test@example.com",
					FirstName: "Test",
					LastName:  "User",
					Password:  "hashedpassword",
				}
				m.On("GetByEmail", "test@example.com").Return(user, nil)
				m.On("PasswordMatches", "wrongpassword", *user).Return(false, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "User not found",
			requestBody: `{
                "email": "nonexistent@example.com",
                "password": "somepassword"
            }`,
			mockSetup: func(m *MockRepository) {
				m.On("GetByEmail", "nonexistent@example.com").Return((*calltypes.User)(nil), errormsg.ErrUserNotExist)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := service.NewRewardService(mockRepo)

			req := httptest.NewRequest(http.MethodPost, "/authenticate", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			svc.Authenticate(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
				assert.Equal(t, "accessToken", cookies[0].Name)
				assert.Equal(t, "refreshToken", cookies[1].Name)
			}

			// Проверяем вызовы mock
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRewardService_GetLeaderboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockSetup      func(*MockRepository)
		expectedStatus int
	}{
		{
			name: "Successful fetch",
			mockSetup: func(m *MockRepository) {
				users := []*calltypes.User{
					{ID: 1, FirstName: "User1", Score: 100},
					{ID: 2, FirstName: "User2", Score: 200},
				}
				m.On("GetAll").Return(users, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Repository error",
			mockSetup: func(m *MockRepository) {
				m.On("GetAll").Return([]*calltypes.User{}, errormsg.ErrRepositoryError)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			tt.mockSetup(mockRepo)

			svc := service.NewRewardService(mockRepo)

			req := httptest.NewRequest(http.MethodGet, "/users/leaderboard", nil)

			rr := httptest.NewRecorder()

			svc.GetLeaderboard(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRewardService_RetrieveOne(t *testing.T) {
	t.Parallel()

	testUser := &calltypes.User{
		ID:        123,
		FirstName: "test",
	}

	tests := []struct {
		name          string
		urlID         string
		repoResponse  *calltypes.User
		repoError     error
		expectedCode  int
		expectedError bool
	}{
		{
			name:          "successful retrieval",
			urlID:         "123",
			repoResponse:  testUser,
			repoError:     nil,
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "invalid ID in URL",
			urlID:         "abc",
			repoResponse:  nil,
			repoError:     nil,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
		{
			name:          "user not found",
			urlID:         "123",
			repoResponse:  nil,
			repoError:     errormsg.ErrUserNotFound,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			if tt.urlID == "123" {
				mockRepo.On("GetOne", 123).Return(tt.repoResponse, tt.repoError)
			}

			svc := &service.RewardService{Repo: mockRepo}

			req, err := http.NewRequest(http.MethodGet, "/users/"+tt.urlID+"/status", nil)
			require.NoError(t, err)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.urlID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			ctx := context.WithValue(req.Context(), userIDKey, "test-user")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			svc.RetrieveOne(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if !tt.expectedError && tt.expectedCode == http.StatusOK {
				var response *calltypes.JSONResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				require.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, "Retrieved one user from the database", response.Message)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRewardService_CompleteTask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		urlID         string
		points        int
		repoError     error
		expectedCode  int
		expectedError bool
	}{
		{
			name:          "successful points addition",
			urlID:         "123",
			points:        100,
			repoError:     nil,
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "invalid ID in URL",
			urlID:         "abc",
			points:        100,
			repoError:     nil,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
		{
			name:          "repository error",
			urlID:         "123",
			points:        100,
			repoError:     errormsg.ErrRepositoryError,
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)

			if tt.urlID == "123" {
				id, _ := strconv.Atoi(tt.urlID)
				mockRepo.On("AddPoints", id, tt.points).Return(tt.repoError)
			}

			svc := &service.RewardService{Repo: mockRepo}

			req, err := http.NewRequest(http.MethodPost, "/users/"+tt.urlID+"/task/complete", nil)
			require.NoError(t, err)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.urlID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			svc.CompleteTask(rr, req, tt.points)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if !tt.expectedError {
				var response calltypes.JSONResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				require.NoError(t, err)
				assert.False(t, response.Error)
				assert.Contains(t, response.Message, "complete task worked for user with id "+tt.urlID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRewardService_RedeemReferrer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		urlID         string
		referrer      string
		setupMock     func(*MockRepository)
		expectedCode  int
		expectedError bool
	}{
		{
			name:     "successful referrer redemption",
			urlID:    "123",
			referrer: "ref123",
			setupMock: func(m *MockRepository) {
				m.On("RedeemReferrer", 123, "ref123").Return(nil)
			},
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "invalid ID",
			urlID:         "abc",
			referrer:      "ref123",
			setupMock:     func(_ *MockRepository) {},
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
		{
			name:     "repository error",
			urlID:    "123",
			referrer: "ref123",
			setupMock: func(m *MockRepository) {
				m.On("RedeemReferrer", 123, "ref123").Return(errormsg.ErrRepositoryError)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)

			svc := &service.RewardService{Repo: mockRepo}

			requestBody := fmt.Sprintf(`{"referrer": "%s"}`, tt.referrer)
			req, err := http.NewRequest(http.MethodPost, "/users/"+tt.urlID+"/referrer", strings.NewReader(requestBody))
			require.NoError(t, err)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.urlID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			svc.RedeemReferrer(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if !tt.expectedError {
				var response calltypes.JSONResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				require.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, "Referrer redeemed", response.Message)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
