package config

import "time"

type Session struct {
	CookieName string        `validate:"required"`
	Lifetime   time.Duration `validate:"required"`
}
