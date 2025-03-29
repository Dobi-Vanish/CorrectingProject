package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	maxConnectionAttempts = 10
	retryDelay            = 2 * time.Second
	srvReadHeaderTimeout  = 5 * time.Second
	srvReadTimeout        = 10 * time.Second
	srvWriteTimeout       = 10 * time.Second
	srvIdleTimeout        = 30 * time.Second
)

//go:embed migrations/*.sql
var EmbedMigrations embed.FS

type Server struct {
	DB        *sql.DB
	Router    *http.ServeMux
	Client    *http.Client
	SecretKey string
}

// NewServer создаёт и настраивает экземпляр сервера.
func NewServer() (*Server, error) {
	if err := godotenv.Load("configs/app.env"); err != nil {
		return nil, fmt.Errorf("loading .env file: %w", err)
	}

	// Подключение к БД
	db, err := connectToDB()
	if err != nil {
		return nil, fmt.Errorf("db connection failed: %w", err)
	}

	// Применение миграций
	if err := applyMigrations(db); err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	return &Server{
		DB:        db,
		Router:    http.NewServeMux(),
		Client:    &http.Client{},
		SecretKey: os.Getenv("SECRET_KEY"),
	}, nil
}

// Start запускает HTTP-сервер.
func (s *Server) Start() error {
	webPort := os.Getenv("PORT")

	srv := &http.Server{
		Addr:              ":" + webPort,
		Handler:           s.Router,
		ReadHeaderTimeout: srvReadHeaderTimeout,
		ReadTimeout:       srvReadTimeout,
		WriteTimeout:      srvWriteTimeout,
		IdleTimeout:       srvIdleTimeout,
	}

	log.Printf("Starting server on port %s", webPort)
	return srv.ListenAndServe()
}

func connectToDB() (*sql.DB, error) {
	dsn := os.Getenv("DSN")

	var counts int

	for {
		db, err := sql.Open("pgx/v4", dsn)
		if err != nil {
			return nil, fmt.Errorf("opening db connection: %w", err)
		}

		if err := db.Ping(); err == nil {
			log.Println("Connected to Postgres!")
			return db, nil
		}

		counts++
		if counts > maxConnectionAttempts {
			return nil, fmt.Errorf("postgres is not ready after %d attempts", counts)
		}

		log.Printf("Postgres not ready (attempt %d/%d), retrying...", counts, maxConnectionAttempts)
		time.Sleep(retryDelay)
	}
}

func applyMigrations(db *sql.DB) error {
	goose.SetBaseFS(EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	migrationsDir := os.Getenv("GOOSE_MIGRATION_DIR")
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}

	return nil
}
