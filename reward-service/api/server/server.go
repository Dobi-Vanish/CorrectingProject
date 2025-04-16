package server

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"log"
	"net/http"
	"reward-service/api/server/router/network"
	"reward-service/internal/postgres/models"
	"reward-service/internal/service"
	"reward-service/migrations"
	"reward-service/pkg/consts"
	"reward-service/pkg/db"
	"reward-service/pkg/errormsg"
	"time"
)

type Server struct {
	cfg    *network.Config
	router *chi.Mux
}

func NewServer(cfg *network.Config) (*Server, error) {
	conn, err := db.Connect(cfg.DB.DSN)
	if err != nil {
		return nil, errormsg.ErrConnectDB
	}

	if err := migrations.Apply(conn); err != nil {
		return nil, errormsg.ErrApplyMigrations
	}

	repo := models.NewPostgresRepository(conn)
	svc := service.NewRewardService(repo)

	router := chi.NewRouter()
	router.Use(network.CORS())
	router.Get("/swagger/*", httpSwagger.WrapHandler)

	handler := network.SetupRoutes(svc)
	router.Mount("/", handler)

	return &Server{
		cfg:    cfg,
		router: router,
	}, nil
}

func (s *Server) Start() error {
	server := &http.Server{
		Addr:         ":" + s.cfg.Server.Port,
		Handler:      s.router,
		ReadTimeout:  consts.ReadTimeout * time.Second,
		WriteTimeout: consts.WriteTimeout * time.Second,
		IdleTimeout:  consts.IdleTimeout * time.Second,
	}

	log.Printf("Server started on :%s", s.cfg.Server.Port)

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}
