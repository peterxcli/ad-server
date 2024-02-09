package model

import "gorm.io/gorm"

type PsychoTest struct {
	gorm.Model
	Type  string `gorm:"type:varchar(255);unique"`
	Count int    `gorm:"type:int"`
}
