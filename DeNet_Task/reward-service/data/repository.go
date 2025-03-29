package data

type Repository interface {
	GetAll() ([]*User, error)
	GetByEmail(email string) (*User, error)
	GetOne(id int) (*User, error)
	Update(user User) error
	DeleteByID(id int) error
	Insert(user User) (int, error)
	PasswordMatches(plainText string, user User) (bool, error)
	AddPoints(id, point int) error
	RedeemReferrer(id int, referrer string) error
	EmailCheck(email string) (*User, error)
	UpdateScore(user User) error
}
