package model

import "time"

type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name" binding:"required"`
	Price     float64   `json:"price" binding:"required,gt=0"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
