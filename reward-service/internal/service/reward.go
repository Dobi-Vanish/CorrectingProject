// Package service implements reward service API handlers.
package service

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"reward-service/api/calltypes"
	"reward-service/api/server/httputils"
	"reward-service/internal/postgres/repository"
	"reward-service/internal/token"
	"reward-service/pkg/consts"
	"reward-service/pkg/errormsg"
	"strconv"
	"strings"
	"time"
)

func NewRewardService(repo repository.Repository) *RewardService {
	return &RewardService{
		Repo:   repo,
		Client: &http.Client{},
	}
}

// GetIDFromURL godoc
// @Summary Extract ID from URL parameter
// @Description Parses and validates ID from URL path
// @Tags Utilities
// @Param paramName path string true "URL parameter name containing ID"
// @Success 200 {integer} int "Valid ID"
// @Failure 400 {object} calltypes.ErrorResponse "Invalid or empty ID"
// @Router /parse-id/{paramName} [get].
func GetIDFromURL(r *http.Request, paramName string) (int, error) {
	idStr := chi.URLParam(r, paramName)
	idStr = strings.TrimSpace(idStr)

	if idStr == "" {
		return 0, errormsg.ErrEmptyID
	}

	for _, c := range idStr {
		if c == '-' && len(idStr) > 1 {
			continue
		}

		if c < '0' || c > '9' {
			return 0, errormsg.ErrInvalidID
		}
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errormsg.ErrInvalidID
	}

	return id, nil
}

// Registrate godoc
// @Summary Register new user
// @Description Creates new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param request body calltypes.RegisterRequest true "User registration data"
// @Success 202 {object} calltypes.JSONResponse
// @Failure 400 {object} calltypes.ErrorResponse "Invalid request data"
// @Router /register [post].
func (s *RewardService) Registrate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Password  string `json:"password"`
		Active    int    `json:"active,omitempty"`
		Score     int    `json:"score,omitempty"`
		Referrer  string `json:"referrer,omitempty"`
	}

	err := httputils.ReadJSON(w, r, &requestPayload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}

	if len(requestPayload.Password) < consts.AtLeastPassLength {
		httputils.ErrorJSON(w, errormsg.ErrPasswordLength, http.StatusBadRequest)

		return
	}

	user := calltypes.User{
		Email:     requestPayload.Email,
		FirstName: requestPayload.FirstName,
		LastName:  requestPayload.LastName,
		Password:  requestPayload.Password,
		Active:    requestPayload.Active,
		Score:     requestPayload.Score,
		Referrer:  requestPayload.Referrer,
	}

	id, err := s.Repo.Insert(user)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}

	payload := calltypes.JSONResponse{
		Error:   false,
		Message: fmt.Sprintf("Successfully created new user, id: %d", id),
	}

	err = httputils.WriteJSON(w, http.StatusAccepted, payload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// GetLeaderboard godoc
// @Summary Get user leaderboard
// @Description Returns all users ordered by score
// @Tags Users
// @Produce json
// @Success 200 {object} calltypes.JSONResponse{data=[]calltypes.User}
// @Failure 400 {object} calltypes.ErrorResponse "Failed to fetch users"
// @Router /leaderboard [get].
func (s *RewardService) GetLeaderboard(w http.ResponseWriter, _ *http.Request) {
	users, err := s.Repo.GetAll()
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrFetchUsers, http.StatusBadRequest)

		return
	}

	payload := calltypes.JSONResponse{
		Error:   false,
		Message: "Fetched all users",
		Data:    users,
	}

	err = httputils.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// Authenticate godoc
// @Summary Authenticate user
// @Description Logs in user and returns auth cookies
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body calltypes.LoginRequest true "Credentials"
// @Success 200 {object} calltypes.JSONResponse
// @Header 200 {string} Set-Cookie "accessToken"
// @Header 200 {string} Set-Cookie "refreshToken"
// @Failure 400 {object} calltypes.ErrorResponse "Invalid credentials"
// @Router /login [post].
func (s *RewardService) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := httputils.ReadJSON(w, r, &requestPayload); err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}

	if requestPayload.Email == "" || requestPayload.Password == "" {
		httputils.ErrorJSON(w, errormsg.ErrUserNotFound, http.StatusBadRequest)

		return
	}

	user, err := s.Repo.GetByEmail(requestPayload.Email)
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrUserNotExist, http.StatusBadRequest)

		return
	}

	valid, err := s.Repo.PasswordMatches(requestPayload.Password, *user)
	if err != nil || !valid {
		httputils.ErrorJSON(w, errormsg.ErrInvalidPassword, http.StatusBadRequest)

		return
	}

	tokenService := token.NewTokenService()

	accessToken, hashedRefreshToken, err := tokenService.GenerateTokens(user.ID)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusInternalServerError)

		return
	}

	err = s.Repo.StoreRefreshToken(user.ID, hashedRefreshToken)
	if err != nil {
		fmt.Println("Error during storing refresh token is: ", err)

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(consts.AccessTokenExpireTime),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    hashedRefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(consts.RefreshTokenExpireTime),
	})

	payload := calltypes.JSONResponse{
		Error:   false,
		Message: fmt.Sprintf("Welcome back, %s!", user.FirstName),
		Data:    map[string]interface{}{"user_id": user.ID},
	}

	err = httputils.WriteJSON(w, http.StatusOK, payload, nil)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// SomeTask godoc
// @Summary Complete some task
// @Description Awards fixed points for some task
// @Tags Tasks
// @Param id path int true "User ID"
// @Success 200 {object} calltypes.JSONResponse
// @Failure 400 {object} calltypes.ErrorResponse "Invalid user ID"
// @Router /tasks/some/{id} [post].
func (s *RewardService) SomeTask(w http.ResponseWriter, r *http.Request) {
	s.CompleteTask(w, r, consts.FixedRewardForSomeTask)
}

// CompleteTask godoc
// @Summary Complete task and earn points
// @Description Awards points to user for completing task
// @Tags Tasks
// @Param id path int true "User ID"
// @Param points query int true "Points to award"
// @Success 200 {object} calltypes.JSONResponse
// @Failure 400 {object} calltypes.ErrorResponse "Failed to add points"
// @Router /tasks/complete/{id} [post].
func (s *RewardService) CompleteTask(w http.ResponseWriter, r *http.Request, points int) {
	id, err := GetIDFromURL(r, "id")
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrInvalidID, http.StatusBadRequest)

		return
	}

	err = s.Repo.AddPoints(id, points)
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrAddPoints, http.StatusBadRequest)

		return
	}

	payload := calltypes.JSONResponse{
		Error:   false,
		Message: fmt.Sprintf("complete task worked for user with id %d, added points %d", id, points),
	}

	err = httputils.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}
}

func (s *RewardService) CompleteTelegramSign(w http.ResponseWriter, r *http.Request) {
	s.CompleteTask(w, r, consts.FixedRewardForTelegramSign)
}

func (s *RewardService) CompleteXSign(w http.ResponseWriter, r *http.Request) {
	s.CompleteTask(w, r, consts.FixedRewardForXSign)
}

// Kuarhodron godoc
// @Summary Secret task endpoint
// @Description Complete secret task with special password
// @Tags Tasks
// @Accept json
// @Param request body calltypes.SecretTaskRequest true "Secret password"
// @Success 200 {object} calltypes.JSONResponse
// @Failure 400 {object} calltypes.ErrorResponse "Invalid password"
// @Router /secret-task [post].
func (s *RewardService) Kuarhodron(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		SecretWaterPassword string `json:"waterPassword"`
	}

	err := httputils.ReadJSON(w, r, &requestPayload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}

	if requestPayload.SecretWaterPassword == "KUARHODRON" {
		s.CompleteTask(w, r, consts.FixedReardForSecretTask)

		return
	}

	httputils.ErrorJSON(w, errormsg.ErrRedeemReferrer, http.StatusBadRequest)
}

// RetrieveOne godoc
// @Summary Get user by ID
// @Description Returns single user data
// @Tags Users
// @Param id path int true "User ID"
// @Produce json
// @Success 200 {object} calltypes.JSONResponse{data=calltypes.User}
// @Failure 400 {object} calltypes.ErrorResponse "User not found"
// @Router /users/{id} [get].
func (s *RewardService) RetrieveOne(w http.ResponseWriter, r *http.Request) {
	id, err := GetIDFromURL(r, "id")
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrInvalidID, http.StatusBadRequest)

		return
	}

	user, err := s.Repo.GetOne(id)
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrFetchUser, http.StatusBadRequest)

		return
	}

	payload := calltypes.JSONResponse{
		Error:   false,
		Message: "Retrieved one user from the database",
		Data:    user,
	}

	err = httputils.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// RedeemReferrer godoc
// @Summary Redeem referrer code
// @Description Applies referrer code to user account
// @Tags Users
// @Accept json
// @Param id path int true "User ID"
// @Param request body calltypes.ReferrerRequest true "Referrer code"
// @Success 200 {object} calltypes.JSONResponse
// @Failure 400 {object} calltypes.ErrorResponse "Invalid referrer code"
// @Router /users/{id}/referrer [post].
func (s *RewardService) RedeemReferrer(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Referrer string `json:"referrer"`
	}

	id, err := GetIDFromURL(r, "id")
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrInvalidID, http.StatusBadRequest)

		return
	}

	err = httputils.ReadJSON(w, r, &requestPayload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}

	err = s.Repo.RedeemReferrer(id, requestPayload.Referrer)
	if err != nil {
		httputils.ErrorJSON(w, errormsg.ErrRedeemReferrer, http.StatusBadRequest)

		return
	}

	payload := calltypes.JSONResponse{
		Error:   false,
		Message: "Referrer redeemed",
	}

	err = httputils.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		httputils.ErrorJSON(w, err, http.StatusBadRequest)

		return
	}
}
