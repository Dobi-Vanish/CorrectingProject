package network

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"reward-service/api/server/middleware"
	"reward-service/internal/service"
)

// SetupRoutes set up the Routes
// @BasePath.
func SetupRoutes(svc *service.RewardService) http.Handler {
	r := chi.NewRouter()

	r.Group(func(secure chi.Router) {
		secure.Use(middleware.Auth())

		secure.Get("/users/{id}/status", svc.RetrieveOne)
		secure.Get("/users/leaderboard", svc.GetLeaderboard)
		secure.Post("/users/{id}/task/telegramSign", svc.CompleteTelegramSign)
		secure.Post("/users/{id}/task/XSign", svc.CompleteXSign)
		secure.Post("/users/{id}/referrer", svc.RedeemReferrer)
		secure.Post("/users/{id}/task/complete", svc.SomeTask)
		secure.Post("/users/{id}/kuarhodron", svc.Kuarhodron)
	})

	r.Post("/authenticate", svc.Authenticate)
	r.Post("/registrate", svc.Registrate)

	return r
}
