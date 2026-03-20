package models

type Workspace struct {
	ID   string `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}
