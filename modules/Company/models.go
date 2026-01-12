package Company

import (
	"gorm.io/gorm"
)

type Company struct {
	gorm.Model
	Name    string  `json:"name" gorm:"not null"`
	Email   string  `json:"email" gorm:"uniqueIndex;not null"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
}

type CreateCompanyRequest struct {
	Name    string  `json:"name" binding:"required"`
	Email   string  `json:"email" binding:"required"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
}

type UpdateCompanyRequest struct {
	Name    *string `json:"name,omitempty"`
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
}
