package entities

import "github.com/google/uuid"

type User struct {
	ID          uuid.UUID `json:"Id"`
	Email       string    `json:"Email"`
	Username    string    `json:"Username"`
	DisplayName string    `json:"DisplayName"`
	AvatarURL   *string   `json:"AvatarUrl,omitempty"`
	Timestamps
}
