package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	dbTimeout     = time.Second * 3
	passMinLength = 8
	bcryptCost    = 12
)

type PostgresRepository struct {
	Conn *sql.DB
}

func NewPostgresRepository(pool *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		Conn: pool,
	}
}

// User is the structure which holds one user from the database.
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Password  string    `json:"-"`
	Active    int       `json:"active,omitempty"`
	Score     int       `json:"score,omitempty"`
	Referrer  string    `json:"referrer,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (u *PostgresRepository) execQuery(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	result, err := u.Conn.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query : %w", err)
	}

	return result, nil
}

func (u *PostgresRepository) queryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	return u.Conn.QueryRowContext(ctx, query, args...)
}

// UserExists проверяет, существует ли пользователь с указанным id.
func (u *PostgresRepository) UserExists(id int) (bool, error) {
	var exists bool

	err := u.queryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		log.Println("failed to check if user exists: ", err)

		return false, fmt.Errorf("failed to check user existence (id: %d): %w", id, err)
	}

	return exists, nil
}

// AddPoints adds  some points.
func (u *PostgresRepository) AddPoints(id, point int) error {
	idExists, err := u.UserExists(id)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return ErrUserNotFound
	}

	stmt := `update users set score = score + $1, updatedAt = $2 where id = $3`

	_, err = u.execQuery(context.Background(), stmt, point, time.Now(), id)
	if err != nil {
		log.Printf("Error adding points to user %d: %v", id, err)

		return ErrAddPointsFailed
	}

	return nil
}

// GetAll returns a slice of all users, sorted by last name.
func (u *PostgresRepository) GetAll() ([]*User, error) {
	query := `select id, email, firstName, lastName, active, score, createdAt, updatedAt, referrer
              from users order by score desc`

	rows, err := u.Conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, ErrFetchUser
	}
	defer rows.Close()

	var users []*User

	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Active,
			&user.Score,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Referrer,
		)

		if err != nil {
			log.Printf("Error scanning user: %v", err)

			return nil, ErrScanUser
		}

		users = append(users, &user)
	}

	return users, nil
}

// EmailCheck using to auth, gets password by provided email.
func (u *PostgresRepository) EmailCheck(email string) (*User, error) {
	var emailExists bool

	err := u.queryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&emailExists)
	if err != nil {
		log.Println("failed to check email: ")

		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	if !emailExists {
		log.Println("User with that email does not exists")

		return nil, fmt.Errorf("User with that email does not exists: %w", err)
	}

	query := `select firstName, password from users where email = $1`

	var user User
	err = u.queryRow(context.Background(), query, email).Scan(
		&user.FirstName,
		&user.Password,
	)

	if err != nil {
		log.Println("failed to fetch user's password by email")

		return nil, fmt.Errorf("failed to fecth user's password by email: %w", err)
	}

	return &user, nil
}

// GetByEmail returns info of one user by email.
func (u *PostgresRepository) GetByEmail(email string) (*User, error) {
	query := `select id, email, firstName, lastName, password, active, score, createdAt, updatedAt 
              from users where email = $1`

	var user User
	err := u.queryRow(context.Background(), query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.Active,
		&user.Score,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		log.Println("failed to fetch user by email")

		return nil, fmt.Errorf("failed to fecth user's password by email: %w", err)
	}

	return &user, nil
}

// RedeemReferrer redeems the referrer with provided id and referrer, adds points to both users.
func (u *PostgresRepository) RedeemReferrer(id int, referrer string) error {
	var referrerExists, idExists bool

	var sameCheck string

	err := u.queryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE referrer = $1)", referrer).Scan(&referrerExists)
	if err != nil {
		log.Println("failed to check referrer")

		return fmt.Errorf("failed to check referrer: %w", err)
	}

	if !referrerExists {
		log.Println("referrer does not exists")

		return fmt.Errorf("referrer does not exists: %w", err)
	}

	idExists, err = u.UserExists(id)
	if err != nil {
		log.Println("User does not exist")

		return fmt.Errorf("user does not exists: %w", err)
	}

	if !idExists {
		log.Println("User does not exist")

		return fmt.Errorf("user does not exists: %w", err)
	}

	err = u.queryRow(context.Background(), "SELECT referrer FROM users WHERE id = $1", id).Scan(&sameCheck)
	if err != nil {
		log.Println("failed to get user's referrer")

		return fmt.Errorf("failed to get user's referrer: %w", err)
	}

	if sameCheck == referrer {
		log.Println("User cannot redeem for their own referrer")

		return fmt.Errorf("User cannot redeem for their own referrer: %w", err)
	}

	_, err = u.execQuery(context.Background(), "UPDATE users SET score = score + 100 WHERE referrer = $1", referrer)
	if err != nil {
		log.Println("failed to update referrer's score")

		return fmt.Errorf("failed to update referrer's score: %w", err)
	}

	_, err = u.execQuery(context.Background(), "UPDATE users SET score = score + 25 WHERE id = $1", id)
	if err != nil {
		log.Println("failed to update score for who redeemed referrer")

		return fmt.Errorf("failed to update score for who redeemed referrer: %w", err)
	}

	return nil
}

// GetOne returns one user by id.
func (u *PostgresRepository) GetOne(id int) (*User, error) {
	idExists, err := u.UserExists(id)
	if err != nil {
		return nil, err
	}

	if !idExists {
		log.Println("User does not exist")

		return nil, ErrUserNotFound
	}

	query := `select id, email, firstName, lastName, active, score, createdAt, updatedAt, referrer 
              from users where id = $1`

	var user User
	err = u.queryRow(context.Background(), query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Active,
		&user.Score,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Referrer,
	)

	if err != nil {
		log.Println("failed to fetch user by id: ", err)

		return nil, fmt.Errorf("failed to fetch user by id: %w", err)
	}

	return &user, nil
}

// Update updates one user in the database, using the information stored in the receiver u.
func (u *PostgresRepository) Update(user User) error {
	idExists, err := u.UserExists(user.ID)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return ErrUserNotFound
	}

	stmt := `update users set
             email = $1,
             firstName = $2,
             lastName = $3,
             active = $4,
             updatedAt = $5
             where id = $6`

	_, err = u.execQuery(context.Background(), stmt,
		user.Email,
		user.FirstName,
		user.LastName,
		user.Active,
		time.Now(),
		user.ID,
	)
	if err != nil {
		log.Println("failed to update user: ", err)

		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateScore provides whole new score to the user.
func (u *PostgresRepository) UpdateScore(user User) error {
	idExists, err := u.UserExists(user.ID)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return ErrUserNotFound
	}

	stmt := `update users set
             score = $1,
             updatedAt = $2
             where id = $3`

	_, err = u.execQuery(context.Background(), stmt,
		user.Score,
		time.Now(),
		user.ID,
	)
	if err != nil {
		log.Println("failed to update user's score: ", err)

		return fmt.Errorf("failed to update user's score: %w", err)
	}

	return nil
}

// DeleteByID deletes one user from the database, by ID.
func (u *PostgresRepository) DeleteByID(id int) error {
	idExists, err := u.UserExists(id)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return ErrUserNotFound
	}

	stmt := `delete from users where id = $1`

	_, err = u.execQuery(context.Background(), stmt, id)
	if err != nil {
		log.Println("failed to delete user by id: ", err)

		return fmt.Errorf("failed to delete user by id: %w", err)
	}

	return nil
}

func (u *PostgresRepository) Insert(user User) (int, error) {
	if len(user.Password) < passMinLength {
		return 0, ErrPasswordLength
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcryptCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	var newID int

	stmt := `insert into users (email, firstName, lastName, password, active, score, createdAt, updatedAt, referrer)
             values ($1, $2, $3, $4, $5, $6, $7, $8, $9) returning id`

	err = u.queryRow(context.Background(), stmt,
		user.Email,
		user.FirstName,
		user.LastName,
		hashedPassword,
		user.Active,
		user.Score,
		time.Now(),
		time.Now(),
		user.Referrer,
	).Scan(&newID)
	if err != nil {
		log.Println("failed to insert new user: ", err)

		return 0, fmt.Errorf("failed to insert new user: %w", err)
	}

	return newID, nil
}

// PasswordMatches uses Go's bcrypt package to compare a user supplied password
// with the hash we have stored for a given user in the database. If the password
// and hash match, we return true; otherwise, we return false.
func (u *PostgresRepository) PasswordMatches(plainText string, user User) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(plainText))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, fmt.Errorf("failed to compare passwords: %w", err)
		}
	}

	return true, nil
}
