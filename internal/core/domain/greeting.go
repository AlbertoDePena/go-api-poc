package domain

import "time"

type Greeting struct {
	ID        string
	Name      string
	Message   string
	CreatedAt time.Time
}

func NewGreeting(id, name string) *Greeting {
	return &Greeting{
		ID:        id,
		Name:      name,
		Message:   "Hello, " + name + "!",
		CreatedAt: time.Now(),
	}
}
