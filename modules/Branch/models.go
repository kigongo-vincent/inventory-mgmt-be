package Branch

import (
	"gorm.io/gorm"
)

type Branch struct {
	gorm.Model
	CompanyID  uint    `json:"companyId" gorm:"not null;index"`
	AdminUserID *uint  `json:"adminUserId,omitempty" gorm:"index"` // Foreign key to User (company admin)
	Name       string  `json:"name" gorm:"not null"`
	Address    *string `json:"address,omitempty"`
	Phone      *string `json:"phone,omitempty"`
}

type CreateBranchRequest struct {
	CompanyID   uint    `json:"companyId" binding:"required"`
	AdminUserID *uint   `json:"adminUserId,omitempty"` // Foreign key to User (company admin)
	Name        string  `json:"name" binding:"required"`
	Address     *string `json:"address,omitempty"`
	Phone       *string `json:"phone,omitempty"`
}

type UpdateBranchRequest struct {
	CompanyID   *uint   `json:"companyId,omitempty"`
	AdminUserID *uint   `json:"adminUserId,omitempty"` // Foreign key to User (company admin)
	Name        *string `json:"name,omitempty"`
	Address     *string `json:"address,omitempty"`
	Phone       *string `json:"phone,omitempty"`
}
