package Expense

import (
	"gorm.io/gorm"
)

type Expense struct {
	gorm.Model
	Amount      float64 `json:"amount" gorm:"not null"`
	Description string  `json:"description" gorm:"not null"`
	Category    string  `json:"category"` // e.g. "Food", "Airtime", "Transport", "Other"
	Currency    string  `json:"currency" gorm:"not null"`
	UserID      uint    `json:"userId" gorm:"not null;index"`
	BranchID    uint    `json:"branchId" gorm:"not null;index"`

	// Relationships (for JSON response - computed from FKs)
	User   *UserResponse   `json:"user,omitempty" gorm:"-"`
	Branch *BranchResponse `json:"branch,omitempty" gorm:"-"`
}

type UserResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type BranchResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type CreateExpenseRequest struct {
	Amount      float64 `json:"amount" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Category    string  `json:"category"`
	Currency    string  `json:"currency" binding:"required"`
	UserID      uint    `json:"userId"`   // Optional - set from token
	BranchID    uint    `json:"branchId"` // Optional - set from user's branch
}

type UpdateExpenseRequest struct {
	Amount      *float64 `json:"amount,omitempty"`
	Description *string  `json:"description,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Currency    *string  `json:"currency,omitempty"`
}
