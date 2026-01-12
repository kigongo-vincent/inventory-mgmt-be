package Product

import (
	"database/sql/driver"
	"encoding/json"

	"gorm.io/gorm"
)

type SyncStatus string

const (
	Online  SyncStatus = "online"
	Offline SyncStatus = "offline"
	Synced  SyncStatus = "synced"
)

// JSONB is a custom type for handling PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), j)
	}

	return json.Unmarshal(bytes, j)
}

type Product struct {
	gorm.Model
	Name       string      `json:"name" gorm:"not null"`
	Price      float64     `json:"price" gorm:"not null"`
	Currency   string      `json:"currency" gorm:"not null"`
	Quantity   int         `json:"quantity" gorm:"not null"`
	ImageURI   *string     `json:"imageUri,omitempty"`
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`
	Attributes JSONB       `json:"attributes" gorm:"type:jsonb"`

	// Foreign Key - Products belong to Company only
	CompanyID uint `json:"companyId" gorm:"not null;index"`

	// Relationship (for JSON response - computed from FK)
	Company *CompanyResponse `json:"company,omitempty" gorm:"-"`
}

// CompanyResponse is used for JSON serialization
type CompanyResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type CreateProductRequest struct {
	Name       string                 `json:"name" binding:"required"`
	Price      float64                `json:"price" binding:"required"`
	Currency   string                 `json:"currency" binding:"required"`
	CompanyID  uint                   `json:"companyId" binding:"required"`
	Quantity   int                    `json:"quantity" binding:"required"`
	ImageURI   *string                `json:"imageUri,omitempty"`
	Attributes map[string]interface{} `json:"attributes"`
}

type UpdateProductRequest struct {
	Name       *string                `json:"name,omitempty"`
	Price      *float64               `json:"price,omitempty"`
	Currency   *string                `json:"currency,omitempty"`
	CompanyID  *uint                  `json:"companyId,omitempty"`
	Quantity   *int                   `json:"quantity,omitempty"`
	ImageURI   *string                `json:"imageUri,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
