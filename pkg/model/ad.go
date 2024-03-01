package model

import (
	"time"

	"github.com/hashicorp/go-memdb"
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

// StartAt < Now() < EndAt
type GetAdRequest struct {
	// AgeStart < Age < AgeEnd
	Age      int
	Country  string
	Gender   string
	Platform string

	Offset int
	Limit  int
}


var schema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		"ad": {
			Name: "ad",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "ID"},
				},
				"country": {
					Name:    "country",
					Unique:  false,
					Indexer: &memdb.StringSliceFieldIndex{Field: "Country"},
				},
				"gender": {
					Name:    "gender",
					Unique:  false,
					Indexer: &memdb.StringSliceFieldIndex{Field: "Gender"},
				},
				"platform": {
					Name:    "platform",
					Unique:  false,
					Indexer: &memdb.StringSliceFieldIndex{Field: "Platform"},
				},
			},
		},
	},
}
