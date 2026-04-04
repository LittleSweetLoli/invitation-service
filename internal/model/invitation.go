package model

import "time"

// Invitation представляет код приглашения с ограниченным количеством использований.
type Invitation struct {
	ID        int64
	Code      string
	MaxUses   int
	CreatedAt time.Time
}

// InvitationUse представляет однократное использование приглашения пользователем.
type InvitationUse struct {
	ID           int64
	InvitationID int64
	Email        string
	CreatedAt    time.Time
}
