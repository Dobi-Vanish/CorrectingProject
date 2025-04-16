package repository

import (
	"reward-service/api/calltypes"
)

type Repository interface {
	GetAll() ([]*calltypes.User, error)
	GetByEmail(email string) (*calltypes.User, error)
	GetOne(id int) (*calltypes.User, error)
	Update(user calltypes.User) error
	Insert(user calltypes.User) (int, error)
	PasswordMatches(plainText string, user calltypes.User) (bool, error)
	AddPoints(id, point int) error
	RedeemReferrer(id int, referrer string) error
	EmailCheck(email string) (*calltypes.User, error)
	UpdateScore(user calltypes.User) error
	StoreRefreshToken(userID int, hashedToken string) error
}
