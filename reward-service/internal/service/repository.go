package service

import (
	"net/http"
	"reward-service/internal/postgres/repository"
)

type RewardServiceInterface interface {
	RetrieveOne(w http.ResponseWriter, r *http.Request)
	GetLeaderboard(w http.ResponseWriter, r *http.Request)
	CompleteTelegramSign(w http.ResponseWriter, r *http.Request)
	CompleteXSign(w http.ResponseWriter, r *http.Request)
	RedeemReferrer(w http.ResponseWriter, r *http.Request)
	SomeTask(w http.ResponseWriter, r *http.Request)
	Kuarhodron(w http.ResponseWriter, r *http.Request)
	Authenticate(w http.ResponseWriter, r *http.Request)
	Registrate(w http.ResponseWriter, r *http.Request)
	CompleteTask(w http.ResponseWriter, r *http.Request, points int)
}

type RewardService struct {
	RewardServiceInterface
	Repo   repository.Repository
	Client *http.Client
}
