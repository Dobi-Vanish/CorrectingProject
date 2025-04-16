package consts

import "time"

const (
	DbTimeout                  = time.Second * 3
	PassMinLength              = 8
	BcryptCost                 = 12
	RefreshTokenExpireTime     = 30 * 24 * time.Hour
	FixedRewardForTelegramSign = 50
	FixedRewardForXSign        = 75
	FixedRewardForSomeTask     = 100
	AtLeastPassLength          = 8
	FixedReardForSecretTask    = 10000
	AccessTokenExpireTime      = 15 * time.Minute
	RefreshTokenLength         = 32
	ConnectAttempts            = 10
	WaitBeforeAttempts         = 2
	MaxAge                     = 300
	Megabyte                   = 1 << 20
	IdleTimeout                = 30
	WriteTimeout               = 10
	ReadTimeout                = 5
)
