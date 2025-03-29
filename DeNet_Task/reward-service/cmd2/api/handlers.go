package main

import (
	"fmt"
	"net/http"
	"os"
	"reward-service/data"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName,omitempty"`
	LastName  string    `json:"lastName,omitempty"`
	Password  string    `json:"-"`
	Active    int       `json:"active"`
	Score     int       `json:"score"`
	Referrer  string    `json:"referrer,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type contextKey string

const (
	fixedRewardForTelegramSign            = 50
	fixedRewardForXSign                   = 75
	fixedRewardForSomeTask                = 100
	cookieTimeExpire                      = 15 * time.Minute
	atLeastPassLength                     = 8
	fixedReardForSecretTask               = 10000
	userIDKey                  contextKey = "userID"
)

// getIDFromRequest gets id from the URL.
func (app *Config) getIDFromRequest(w http.ResponseWriter, r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		app.errorJSON(w, ErrConvertID, http.StatusBadRequest)

		return 0, fmt.Errorf("failed to convert ID param '%s' to int: %w", idStr, err)
	}

	return id, nil
}

// Registrate insert new user to the database.
func (app *Config) Registrate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Password  string `json:"password"`
		Active    int    `json:"active,omitempty"`
		Score     int    `json:"score,omitempty"`
		Referrer  string `json:"referrer,omitempty"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}

	if len(requestPayload.Password) < atLeastPassLength {
		app.errorJSON(w, ErrPasswordLength, http.StatusBadRequest)

		return
	}

	user := User{
		Email:     requestPayload.Email,
		FirstName: requestPayload.FirstName,
		LastName:  requestPayload.LastName,
		Password:  requestPayload.Password,
		Active:    requestPayload.Active,
		Score:     requestPayload.Score,
		Referrer:  requestPayload.Referrer,
	}

	id, err := app.Repo.Insert(data.User(user))
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Successfully created new user, id: %d", id),
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// GetLeaderboard retrieves all users from the database, sort them by points.
func (app *Config) GetLeaderboard(w http.ResponseWriter, _ *http.Request) {
	users, err := app.Repo.GetAll()
	if err != nil {
		app.errorJSON(w, ErrFetchUsers, http.StatusBadRequest)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Fetched all users",
		Data:    users,
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// Authenticate authenticates user by provided email and password, provides tokens to access.
func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}

	user, err := app.Repo.EmailCheck(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, ErrUserNotExist, http.StatusBadRequest)

		return
	}

	valid, err := app.Repo.PasswordMatches(requestPayload.Password, *user)
	if err != nil || !valid {
		app.errorJSON(w, ErrInvalidPassword, http.StatusBadRequest)

		return
	}

	secretKey := os.Getenv("SECRET_KEY")
	userData, err := generateTokens(user.ID, secretKey)

	if err != nil {
		app.errorJSON(w, err, http.StatusInternalServerError)

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    userData.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(cookieTimeExpire),
	})

	err = validateRefreshToken(userData.HashedRefreshToken, userData.RefreshToken)
	if err != nil {
		app.errorJSON(w, err, http.StatusInternalServerError)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Welcome back, %s!", user.FirstName),
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	err = app.writeJSON(w, http.StatusAccepted, payload, headers)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
	}
}

// someTask some blank task.
func (app *Config) someTask(w http.ResponseWriter, r *http.Request) {
	app.completeTask(w, r, fixedRewardForSomeTask)
}

// completeTask completes various task and adding some point to the user.
func (app *Config) completeTask(w http.ResponseWriter, r *http.Request, points int) {
	id, err := app.getIDFromRequest(w, r)
	if err != nil {
		return
	}

	err = app.Repo.AddPoints(id, points)
	if err != nil {
		app.errorJSON(w, ErrAddPoints, http.StatusBadRequest)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("complete task worked for user with id %d, added points %d", id, points),
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// completeTelegramSign completes telegram sign to add points to the user.
func (app *Config) completeTelegramSign(w http.ResponseWriter, r *http.Request) {
	app.completeTask(w, r, fixedRewardForTelegramSign)
}

// completeTelegramSign completes X sign to add points to the user.
func (app *Config) completeXSign(w http.ResponseWriter, r *http.Request) {
	app.completeTask(w, r, fixedRewardForXSign)
}

// Kuarhodron special task to add 10k points.
func (app *Config) Kuarhodron(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		SecretWaterPassword string `json:"waterPassword"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}

	if requestPayload.SecretWaterPassword == "KUARHODRON" {
		app.completeTask(w, r, fixedReardForSecretTask)
	}
}

// retrieveOne retrieves one user from the database by id.
func (app *Config) retrieveOne(w http.ResponseWriter, r *http.Request) {
	id, err := app.getIDFromRequest(w, r)
	if err != nil {
		return
	}

	user, err := app.Repo.GetOne(id)
	if err != nil {
		app.errorJSON(w, ErrFetchUser, http.StatusBadRequest)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Retrieved one user from the database",
		Data:    user,
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// redeemReferrer redeems referrer for the owner of the referrer and for the user, who used it base on id and referrer.
func (app *Config) redeemReferrer(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Referrer string `json:"referrer"`
	}

	id, err := app.getIDFromRequest(w, r)
	if err != nil {
		return
	}

	err = app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}

	err = app.Repo.RedeemReferrer(id, requestPayload.Referrer)
	if err != nil {
		app.errorJSON(w, ErrRedeemReferrer, http.StatusBadRequest)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Referrer redeemed",
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}
}

// DeleteUser delets user from the DB.
func (app *Config) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}

	err = app.Repo.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, ErrDeleteUser, http.StatusBadRequest)

		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "User deleted successfully",
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)

		return
	}
}
