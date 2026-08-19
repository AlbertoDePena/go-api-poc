package domain

import "errors"

var (
	ErrGreetingNotFound = errors.New("greeting not found")
	ErrNameRequired     = errors.New("name is required")
)
