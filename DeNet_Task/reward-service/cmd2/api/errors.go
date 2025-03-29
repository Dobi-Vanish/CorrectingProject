package main

import "errors"

// Общие ошибки.
var (
	ErrConvertID       = errors.New("couldn't convert id string to int")
	ErrPasswordLength  = errors.New("password must be at least 8 characters long")
	ErrFetchUsers      = errors.New("couldn't fetch all users")
	ErrUserNotExist    = errors.New("user with this email does not exist")
	ErrInvalidPassword = errors.New("invalid password")
	ErrAddPoints       = errors.New("couldn't add points to the user")
	ErrFetchUser       = errors.New("couldn't fetch user")
	ErrRedeemReferrer  = errors.New("couldn't redeem referrer")
	ErrDeleteUser      = errors.New("couldn't delete user")
	ErrSingleJSON      = errors.New("body must have only a single JSON value")
)
