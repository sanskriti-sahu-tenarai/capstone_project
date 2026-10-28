package model

import (
	"errors"
	"time"
)

type OrderStatus string

const (
	StatusCreated   OrderStatus = "CREATED"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case StatusCreated, StatusConfirmed, StatusCancelled:
		return true
	default:
		return false
	}
}

type Order struct {
	ID         int64       `json:"id" example:"1"`
	CustomerID int64       `json:"customerId" example:"101"`
	ProductID  int64       `json:"productId" example:"501"`
	Quantity   int         `json:"quantity" example:"2"`
	TotalPrice float64     `json:"totalPrice" example:"100.00"`
	Status     OrderStatus `json:"status" example:"CREATED"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type CreateOrderRequest struct {
	CustomerID int64 `json:"customerId" binding:"required,gt=0" example:"101"`
	ProductID  int64 `json:"productId" binding:"required,gt=0" example:"501"`
	Quantity   int   `json:"quantity" binding:"required,gt=0" example:"2"`
}

type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status" binding:"required" example:"CONFIRMED"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"admin123"`
}

type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOi..."`
	Role  string `json:"role" example:"admin"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request input"`
}

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidStatus = errors.New("invalid status transition")
)
