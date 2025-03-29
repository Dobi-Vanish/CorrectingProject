package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reward-service/data"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
	data.PostgresTestRepository
}

type MockConfig struct {
	mock.Mock
	Config
}

func (m *MockRepository) Insert(user data.User) (int, error) {
	args := m.Called(user)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) AddPoints(id, points int) error {
	args := m.Called(id, points)
	return args.Error(0)
}

func (m *MockRepository) RedeemReferrer(id int, referrer string) error {
	args := m.Called(id, referrer)
	return args.Error(0)
}

func (m *MockRepository) DeleteByID(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockRepository) GetOne(id int) (*data.User, error) {
	args := m.Called(id)
	return args.Get(0).(*data.User), args.Error(1)
}

func (m *MockRepository) GetAll() ([]*data.User, error) {
	args := m.Called()
	return args.Get(0).([]*data.User), args.Error(1)
}

func (m *MockRepository) EmailCheck(email string) (*data.User, error) {
	args := m.Called(email)
	return args.Get(0).(*data.User), args.Error(1)
}

func (m *MockRepository) PasswordMatches(password string, user data.User) (bool, error) {
	args := m.Called(password, user)
	return args.Bool(0), args.Error(1)
}

func (m *MockConfig) writeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	args := m.Called(w, status, data, headers)
	return args.Error(0)
}

func TestGetIDFromRequest(t *testing.T) {
	app := &Config{}

	tests := []struct {
		name       string
		url        string
		wantID     int
		wantErr    bool
		wantStatus int
	}{
		{
			name:    "valid id",
			url:     "/users/123",
			wantID:  123,
			wantErr: false,
		},
		{
			name:       "invalid id",
			url:        "/users/abc",
			wantID:     0,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty id",
			url:        "/users/",
			wantID:     0,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()

			// Создаем роутер и добавляем route с параметром
			r := chi.NewRouter()
			r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
				id, err := app.getIDFromRequest(w, r)
				if err != nil {
					return
				}
				if id != tt.wantID {
					t.Errorf("expected id %d, got %d", tt.wantID, id)
				}
			})

			r.ServeHTTP(rr, req)

			if tt.wantErr {
				assert.Equal(t, tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestRegistrate(t *testing.T) {
	mockRepo := new(MockRepository)
	app := &Config{Repo: mockRepo}

	tests := []struct {
		name           string
		input          string
		mockReturnID   int
		mockError      error
		wantStatus     int
		wantErr        bool
		passwordLength int
	}{
		{
			name: "successful registration",
			input: `{
				"email": "test@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"password": "longenoughpassword",
				"active": 1,
				"score": 0,
				"referrer": "ref123"
			}`,
			mockReturnID:   1,
			mockError:      nil,
			wantStatus:     http.StatusAccepted,
			wantErr:        false,
			passwordLength: 10,
		},
		{
			name: "short password",
			input: `{
				"email": "test@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"password": "short",
				"active": 1
			}`,
			wantStatus:     http.StatusBadRequest,
			wantErr:        true,
			passwordLength: 4,
		},
		{
			name: "invalid json",
			input: `{
				"email": "test@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"password": "longenoughpassword",
				"active": "invalid"  // should be int
			}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "repository error",
			input: `{
				"email": "test@example.com",
				"firstName": "John",
				"lastName": "Doe",
				"password": "longenoughpassword",
				"active": 1
			}`,
			mockError:  assert.AnError,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настройка мок репозитория
			mockRepo.On("Insert", mock.Anything).Return(tt.mockReturnID, tt.mockError).Once()

			req, _ := http.NewRequest("POST", "/register", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// Вызов тестируемого метода
			app.Registrate(rr, req)

			// Проверки
			assert.Equal(t, tt.wantStatus, rr.Code)

			if !tt.wantErr {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Contains(t, response.Message, fmt.Sprintf("id: %d", tt.mockReturnID))
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetLeaderboard(t *testing.T) {
	// Создаем мок репозитория
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	tests := []struct {
		name        string
		mockUsers   []data.User
		mockError   error
		wantStatus  int
		wantMessage string
	}{
		{
			name: "successful fetch",
			mockUsers: []data.User{
				{ID: 1, Email: "user1@test.com", Score: 100},
				{ID: 2, Email: "user2@test.com", Score: 200},
			},
			wantStatus:  http.StatusAccepted,
			wantMessage: "Fetched all users",
		},
		{
			name:        "empty leaderboard",
			mockUsers:   []data.User{},
			wantStatus:  http.StatusAccepted,
			wantMessage: "Fetched all users",
		},
		{
			name:        "database error",
			mockError:   fmt.Errorf("database error"),
			wantStatus:  http.StatusBadRequest,
			wantMessage: "couldn't fetch all users",
		},
		{
			name: "sorting by score",
			mockUsers: []data.User{
				{ID: 1, Score: 300},
				{ID: 2, Score: 100},
				{ID: 3, Score: 200},
			},
			wantStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем мок
			mockRepo.On("GetAll").Return(tt.mockUsers, tt.mockError).Once()

			req := httptest.NewRequest("GET", "/leaderboard", nil)
			rr := httptest.NewRecorder()

			app.GetLeaderboard(rr, req)

			// Проверяем статус код
			assert.Equal(t, tt.wantStatus, rr.Code)

			if tt.mockError == nil {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, tt.wantMessage, response.Message)
				assert.Equal(t, tt.mockUsers, response.Data)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthenticate(t *testing.T) {
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	// Устанавливаем SECRET_KEY для тестов
	os.Setenv("SECRET_KEY", "test_secret_key")
	defer os.Unsetenv("SECRET_KEY")

	testUser := &data.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Password:  "hashed_password",
	}

	tests := []struct {
		name          string
		input         string
		mockUser      *data.User
		mockUserError error
		passwordValid bool
		passwordError error
		wantStatus    int
		wantMessage   string
		wantCookie    bool
	}{
		{
			name:          "successful authentication",
			input:         `{"email": "test@example.com", "password": "correct_password"}`,
			mockUser:      testUser,
			passwordValid: true,
			wantStatus:    http.StatusAccepted,
			wantMessage:   "Welcome back, Test!",
			wantCookie:    true,
		},
		{
			name:          "invalid email",
			input:         `{"email": "wrong@example.com", "password": "password"}`,
			mockUserError: data.ErrNoRecord,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "invalid password",
			input:         `{"email": "test@example.com", "password": "wrong_password"}`,
			mockUser:      testUser,
			passwordValid: false,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			input:      `{"email": "test@example.com", "password": 123}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "token generation error",
			input:         `{"email": "test@example.com", "password": "correct_password"}`,
			mockUser:      testUser,
			passwordValid: true,
			wantStatus:    http.StatusInternalServerError,
		},
		{
			name:       "empty email or password",
			input:      `{"email": "", "password": ""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "password check error",
			input:         `{"email": "test@example.com", "password": "password"}`,
			mockUser:      testUser,
			passwordValid: false,
			passwordError: fmt.Errorf("hash error"),
			wantStatus:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем моки
			mockRepo.On("EmailCheck", mock.Anything).Return(tt.mockUser, tt.mockUserError).Once()

			if tt.mockUserError == nil {
				mockRepo.On("PasswordMatches", mock.Anything, *tt.mockUser).Return(tt.passwordValid, tt.passwordError).Once()
			}

			req := httptest.NewRequest("POST", "/authenticate", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			app.Authenticate(rr, req)

			// Проверки
			assert.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantStatus == http.StatusAccepted {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, tt.wantMessage, response.Message)

				// Проверяем cookie
				if tt.wantCookie {
					cookies := rr.Result().Cookies()
					assert.NotEmpty(t, cookies)
					assert.Equal(t, "accessToken", cookies[0].Name)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCompleteTask(t *testing.T) {
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	tests := []struct {
		name          string
		url           string
		points        int
		mockID        int
		mockIDError   error
		mockAddError  error
		wantStatus    int
		wantMessage   string
		wantErrInJson bool
	}{
		{
			name:        "successful task completion",
			url:         "/complete/123",
			points:      100,
			mockID:      123,
			wantStatus:  http.StatusAccepted,
			wantMessage: "complete task worked for user with id 123, added points 100",
		},
		{
			name:          "invalid user id",
			url:           "/complete/abc",
			points:        100,
			mockIDError:   fmt.Errorf("invalid id"),
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "add points error",
			url:           "/complete/123",
			points:        100,
			mockID:        123,
			mockAddError:  fmt.Errorf("database error"),
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "json write error",
			url:           "/complete/123",
			points:        100,
			mockID:        123,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем моки
			mockRepo.On("AddPoints", tt.mockID, tt.points).Return(tt.mockAddError).Maybe()

			// Создаем запрос с chi router context
			req := httptest.NewRequest("POST", tt.url, nil)
			rr := httptest.NewRecorder()

			// Создаем chi router и добавляем параметр id
			r := chi.NewRouter()
			r.Post("/complete/{id}", func(w http.ResponseWriter, r *http.Request) {
				// Подменяем getIDFromRequest для тестов
				if tt.mockIDError != nil {
					app.errorJSON(w, tt.mockIDError, http.StatusBadRequest)
					return
				}
				app.completeTask(w, r, tt.points)
			})

			r.ServeHTTP(rr, req)

			// Проверки
			assert.Equal(t, tt.wantStatus, rr.Code)

			if !tt.wantErrInJson {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, tt.wantMessage, response.Message)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestKuarhodron(t *testing.T) {
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	tests := []struct {
		name               string
		input              string
		wantCompleteCalled bool
		wantStatus         int
	}{
		{
			name:               "correct password",
			input:              `{"waterPassword": "KUARHODRON"}`,
			wantCompleteCalled: true,
			wantStatus:         http.StatusAccepted,
		},
		{
			name:               "incorrect password",
			input:              `{"waterPassword": "wrong"}`,
			wantCompleteCalled: false,
			wantStatus:         http.StatusBadRequest,
		},
		{
			name:               "empty password",
			input:              `{"waterPassword": ""}`,
			wantCompleteCalled: false,
			wantStatus:         http.StatusBadRequest,
		},
		{
			name:               "invalid json",
			input:              `{"waterPassword": 123}`,
			wantCompleteCalled: false,
			wantStatus:         http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Мокируем completeTask
			completeCalled := false
			oldCompleteTask := app.completeTask
			app.completeTask = func(w http.ResponseWriter, r *http.Request, points int) {
				completeCalled = true
				assert.Equal(t, fixedReardForSecretTask, points)
				oldCompleteTask(w, r, points)
			}
			defer func() { app.completeTask = oldCompleteTask }()

			req := httptest.NewRequest("POST", "/kuarhodron", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			app.Kuarhodron(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Equal(t, tt.wantCompleteCalled, completeCalled)
		})
	}
}

func TestRetrieveOne(t *testing.T) {
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	testUser := &data.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Score:     100,
	}

	tests := []struct {
		name          string
		url           string
		mockUser      *data.User
		mockError     error
		wantStatus    int
		wantUser      *data.User
		wantErrInJson bool
	}{
		{
			name:       "successful retrieval",
			url:        "/users/1",
			mockUser:   testUser,
			wantStatus: http.StatusAccepted,
			wantUser:   testUser,
		},
		{
			name:          "user not found",
			url:           "/users/999",
			mockError:     data.ErrNoRecord,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "invalid id",
			url:           "/users/abc",
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "database error",
			url:           "/users/1",
			mockError:     errors.New("database error"),
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем моки
			if !strings.Contains(tt.url, "abc") { // Не мокируем для неверного ID
				mockRepo.On("GetOne", mock.Anything).Return(tt.mockUser, tt.mockError).Once()
			}

			req := httptest.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()

			// Создаем chi router для обработки параметра
			r := chi.NewRouter()
			r.Get("/users/{id}", app.retrieveOne)
			r.ServeHTTP(rr, req)

			// Проверки
			assert.Equal(t, tt.wantStatus, rr.Code)

			if !tt.wantErrInJson && tt.wantUser != nil {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, "Retrieved one user from the database", response.Message)
				assert.Equal(t, tt.wantUser, response.Data)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRedeemReferrer(t *testing.T) {
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	tests := []struct {
		name          string
		url           string
		input         string
		mockID        int
		mockError     error
		wantStatus    int
		wantMessage   string
		wantErrInJson bool
	}{
		{
			name:        "successful redemption",
			url:         "/users/123/referrer",
			input:       `{"referrer": "ref123"}`,
			mockID:      123,
			wantStatus:  http.StatusAccepted,
			wantMessage: "Referrer redeemed",
		},
		{
			name:          "invalid user id",
			url:           "/users/abc/referrer",
			input:         `{"referrer": "ref123"}`,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "empty referrer",
			url:           "/users/123/referrer",
			input:         `{"referrer": ""}`,
			mockID:        123,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "database error",
			url:           "/users/123/referrer",
			input:         `{"referrer": "ref123"}`,
			mockID:        123,
			mockError:     errors.New("database error"),
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "invalid json",
			url:           "/users/123/referrer",
			input:         `{"referrer": 123}`,
			mockID:        123,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем моки
			if tt.mockID != 0 {
				mockRepo.On("RedeemReferrer", tt.mockID, mock.AnythingOfType("string")).Return(tt.mockError).Once()
			}

			req := httptest.NewRequest("POST", tt.url, strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// Создаем chi router для обработки параметра
			r := chi.NewRouter()
			r.Post("/users/{id}/referrer", app.redeemReferrer)
			r.ServeHTTP(rr, req)

			// Проверки
			assert.Equal(t, tt.wantStatus, rr.Code)

			if !tt.wantErrInJson && tt.wantMessage != "" {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, tt.wantMessage, response.Message)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	mockRepo := &MockRepository{}
	app := &Config{Repo: mockRepo}

	tests := []struct {
		name          string
		input         string
		mockError     error
		wantStatus    int
		wantMessage   string
		wantErrInJson bool
	}{
		{
			name:        "successful deletion",
			input:       `{"id": 123}`,
			wantStatus:  http.StatusAccepted,
			wantMessage: "User deleted successfully",
		},
		{
			name:          "invalid id",
			input:         `{"id": "abc"}`,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "database error",
			input:         `{"id": 123}`,
			mockError:     errors.New("database error"),
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "user not found",
			input:         `{"id": 999}`,
			mockError:     data.ErrNoRecord,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
		{
			name:          "empty json",
			input:         `{}`,
			wantStatus:    http.StatusBadRequest,
			wantErrInJson: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем моки
			if strings.Contains(tt.input, `"id": 123`) || strings.Contains(tt.input, `"id": 999`) {
				mockRepo.On("DeleteByID", mock.AnythingOfType("int")).Return(tt.mockError).Once()
			}

			req := httptest.NewRequest("DELETE", "/users", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			app.DeleteUser(rr, req)

			// Проверки
			assert.Equal(t, tt.wantStatus, rr.Code)

			if !tt.wantErrInJson && tt.wantMessage != "" {
				var response jsonResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.False(t, response.Error)
				assert.Equal(t, tt.wantMessage, response.Message)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
