package data

import (
	"database/sql"
)

type PostgresTestRepository struct {
	Conn *sql.DB
}

func NewPostgresTestRepository(db *sql.DB) *PostgresTestRepository {
	return &PostgresTestRepository{
		Conn: db,
	}
}

func (u *PostgresTestRepository) AddPoints(id, point int) error {
	return nil
}

func (u *PostgresTestRepository) GetAll() ([]*User, error) {
	return nil, nil
}

func (u *PostgresTestRepository) EmailCheck(email string) (*User, error) {
	return nil, nil
}

func (u *PostgresTestRepository) GetByEmail(email string) (*User, error) {
	return nil, nil
}

func (u *PostgresTestRepository) RedeemReferrer(id int, referrer string) error {
	return nil
}

func (u *PostgresTestRepository) GetOne(id int) (*User, error) {
	return nil, nil
}

func (u *PostgresTestRepository) Update(user User) error {
	return nil
}

func (u *PostgresTestRepository) UpdateScore(user User) error {
	return nil
}

func (u *PostgresTestRepository) DeleteByID(id int) error {
	return nil
}

func (u *PostgresTestRepository) Insert(user User) (int, error) {
	return 0, nil
}

func (u *PostgresTestRepository) PasswordMatches(plainText string, user User) (bool, error) {
	return true, nil
}
