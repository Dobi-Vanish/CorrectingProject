package data

import "errors"

var (
	ErrUserNotFound    = errors.New("user does not exist")
	ErrAddPointsFailed = errors.New("failed to add points")
	ErrFetchUser       = errors.New("failed to fetch user")
	ErrScanUser        = errors.New("failed to scan user")
	ErrPasswordLength  = errors.New("password must be at least 8 characters long")
	ErrNoRecord        = errors.New("no record provided")
)
