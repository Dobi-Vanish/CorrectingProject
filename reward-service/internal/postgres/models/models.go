package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reward-service/api/calltypes"
	"reward-service/pkg/consts"
	"time"

	"golang.org/x/crypto/bcrypt"
	"reward-service/pkg/errormsg"
)

type PostgresRepository struct {
	Conn *sql.DB
}

func NewPostgresRepository(pool *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		Conn: pool,
	}
}

// UserExists checks does user really exist.
func (u *PostgresRepository) UserExists(id int) (bool, error) {
	var exists bool

	err := u.queryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		log.Println("failed to check if user exists: ", err)

		return false, fmt.Errorf("failed to check user existence (id: %d): %w", id, err)
	}

	return exists, nil
}

// AddPoints adds some points.
func (u *PostgresRepository) AddPoints(id, point int) error {
	idExists, err := u.UserExists(id)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return errormsg.ErrUserNotFound
	}

	stmt := `update users set score = score + $1, updated_at = $2 where id = $3`

	_, err = u.execQuery(context.Background(), stmt, point, time.Now(), id)
	if err != nil {
		log.Printf("Error adding points to user %d: %v", id, err)

		return errormsg.ErrAddPointsFailed
	}

	return nil
}

// GetAll returns a slice of all users, sorted by last name.
func (u *PostgresRepository) GetAll() ([]*calltypes.User, error) {
	query := `select id, email, first_name, last_name, active, score, created_at, updated_at, referrer
              from users order by score desc`

	rows, err := u.Conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, errormsg.ErrFetchUser
	}
	defer rows.Close()

	var users []*calltypes.User

	for rows.Next() {
		var user calltypes.User
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

			return nil, errormsg.ErrScanUser
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error after row iteration: %v", err)

		return nil, errormsg.ErrFetchUser
	}

	return users, nil
}

// EmailCheck using to auth, gets password by provided email.
func (u *PostgresRepository) EmailCheck(email string) (*calltypes.User, error) {
	var emailExists bool

	err := u.queryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&emailExists)
	if err != nil {
		log.Println("failed to check email: ")

		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	if !emailExists {
		log.Println("User with that email does not exists")

		return nil, fmt.Errorf("user with that email does not exists: %w", err)
	}

	query := `select first_name, password from users where email = $1`

	var user calltypes.User
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
func (u *PostgresRepository) GetByEmail(email string) (*calltypes.User, error) {
	query := `select id, email, first_name, last_name, password, active, score, created_at, updated_at 
              from users where email = $1`

	var user calltypes.User
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
		return fmt.Errorf("err occurred during executing query row: %w", err)
	}

	if !referrerExists {
		return fmt.Errorf("provided referrer does not exist: %w", err)
	}

	idExists, err = u.UserExists(id)
	if err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}

	if !idExists {
		return fmt.Errorf("user does not exist: %w", err)
	}

	err = u.queryRow(context.Background(), "SELECT referrer FROM users WHERE id = $1", id).Scan(&sameCheck)
	if err != nil {
		return fmt.Errorf("error finding the referrer: %w", err)
	}

	if sameCheck == referrer {
		return fmt.Errorf("user cannot redeem for their own referrer: %w", err)
	}

	_, err = u.execQuery(context.Background(), "UPDATE users SET score = score + 100 WHERE referrer = $1", referrer)
	if err != nil {
		return fmt.Errorf("failed to update referrer's score: %w", err)
	}

	_, err = u.execQuery(context.Background(), "UPDATE users SET score = score + 25 WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to update score for who redeemed referrer: %w", err)
	}

	return nil
}

// GetOne returns one user by id.
func (u *PostgresRepository) GetOne(id int) (*calltypes.User, error) {
	idExists, err := u.UserExists(id)
	if err != nil {
		return nil, err
	}

	if !idExists {
		log.Println("User does not exist")

		return nil, errormsg.ErrUserNotFound
	}

	query := `select id, email, first_name, last_name, active, score, created_at, updated_at, referrer 
              from users where id = $1`

	var user calltypes.User
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
func (u *PostgresRepository) Update(user calltypes.User) error {
	idExists, err := u.UserExists(user.ID)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return errormsg.ErrUserNotFound
	}

	stmt := `update users set
             email = $1,
             first_name = $2,
             last_name = $3,
             active = $4,
             updated_at = $5
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
func (u *PostgresRepository) UpdateScore(user calltypes.User) error {
	idExists, err := u.UserExists(user.ID)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return errormsg.ErrUserNotFound
	}

	stmt := `update users set
             score = $1,
             updated_at = $2
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

// Insert adds new user to the database.
func (u *PostgresRepository) Insert(user calltypes.User) (int, error) {
	if len(user.Password) < consts.PassMinLength {
		return 0, errormsg.ErrPasswordLength
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), consts.BcryptCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	var newID int

	stmt := `insert into users (email, first_name, last_name, password, active, score, created_at, updated_at, referrer)
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
func (u *PostgresRepository) PasswordMatches(plainText string, user calltypes.User) (bool, error) {
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

// StoreRefreshToken stores provided refresh token.
func (u *PostgresRepository) StoreRefreshToken(userID int, hashedToken string) error {
	idExists, err := u.UserExists(userID)
	if err != nil {
		return err
	}

	if !idExists {
		log.Println("User does not exist")

		return errormsg.ErrUserNotFound
	}

	stmt := `UPDATE users SET refresh_token = $1, refresh_token_expires = $2 WHERE id = $3`

	_, err = u.execQuery(context.Background(), stmt,
		hashedToken,
		time.Now().Add(consts.RefreshTokenExpireTime),
		userID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (u *PostgresRepository) execQuery(ctx context.Context, query string, args ...interface{}) (sql.Result, error) { //nolint: unparam
	ctx, cancel := context.WithTimeout(ctx, consts.DbTimeout)
	defer cancel()

	result, err := u.Conn.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query : %w", err)
	}

	return result, nil
}

func (u *PostgresRepository) queryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(ctx, consts.DbTimeout)
	defer cancel()

	return u.Conn.QueryRowContext(ctx, query, args...)
}
