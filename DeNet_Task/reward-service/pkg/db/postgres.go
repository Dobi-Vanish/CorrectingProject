package db

import (
	"database/sql"
	"log"
	"reward-service/pkg/consts"
	"reward-service/pkg/errormsg"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib" // connection driver.
)

// Connect connects to Postgres.
func Connect(dsn string) (*sql.DB, error) {
	var counts int

	var conn *sql.DB

	for {
		var err error

		conn, err = sql.Open("pgx/v4", dsn)
		if err != nil {
			log.Printf("Postgres not ready (attempt %d): %v", counts, err)

			counts++
		} else {
			if err := conn.Ping(); err == nil {
				log.Println("Connected to Postgres!")

				return conn, nil
			}
		}

		if counts > consts.ConnectAttempts {
			return nil, errormsg.ErrPostgresConnectAttemptsFailed
		}

		time.Sleep(consts.WaitBeforeAttempts * time.Second)
	}
}
