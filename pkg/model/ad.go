package model

import (
	"time"
)

type Ad struct {
	ID       string
	Title    string
	Content  string
	StartAt  time.Time
	EndAt    time.Time
	AgeStart int
	AgeEnd   int
	Gender   []string
	Country  []string
	Platform []string
	// Version is used to handle optimistic lock
	Version int
}
