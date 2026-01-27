package Sale

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type PaymentStatus string

const (
	Credit   PaymentStatus = "credit"
	Promised PaymentStatus = "promised"
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

type Sale struct {
	gorm.Model
	ProductID         uint           `json:"productId" gorm:"not null;index"`
	ProductName       string         `json:"productName" gorm:"not null"`
	ProductAttributes JSONB          `json:"productAttributes" gorm:"type:jsonb"`
	Quantity          int            `json:"quantity" gorm:"not null"`
	UnitPrice         float64        `json:"unitPrice" gorm:"not null"`
	ExtraCosts        float64        `json:"extraCosts" gorm:"default:0"` // Additional costs like delivery charges
	TotalPrice        float64        `json:"totalPrice" gorm:"not null"`
	Currency          string         `json:"currency" gorm:"not null"`
	SellerID          uint           `json:"sellerId" gorm:"not null;index"`
	PaymentStatus     PaymentStatus  `json:"paymentStatus" gorm:"not null"`
	// Buyer information (optional)
	BuyerName     *string       `json:"buyerName,omitempty"`
	BuyerContact  *string       `json:"buyerContact,omitempty"`
	BuyerLocation *string       `json:"buyerLocation,omitempty"`
	SyncStatus    *SyncStatus   `json:"syncStatus,omitempty"`

	// Relationships (for JSON response - computed from FKs)
	Product *ProductResponse `json:"product,omitempty" gorm:"-"`
	Seller  *UserResponse    `json:"seller,omitempty" gorm:"-"`
	Branch  *BranchResponse  `json:"branch,omitempty" gorm:"-"` // Computed from Seller's BranchID
}

// ProductResponse is used for JSON serialization
type ProductResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// UserResponse is used for JSON serialization
type UserResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// BranchResponse is used for JSON serialization
type BranchResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type CreateSaleRequest struct {
	ProductID         uint                   `json:"productId" binding:"required"`
	ProductName        string                 `json:"productName" binding:"required"`
	ProductAttributes  map[string]interface{} `json:"productAttributes"`
	Quantity           int                    `json:"quantity" binding:"required"`
	UnitPrice          float64                `json:"unitPrice" binding:"required"`
	ExtraCosts         float64                `json:"extraCosts"` // Optional additional costs like delivery charges
	TotalPrice         float64                `json:"totalPrice" binding:"required"`
	Currency           string                 `json:"currency" binding:"required"`
	SellerID           uint                   `json:"sellerId"` // Optional - will be set from token context
	PaymentStatus      PaymentStatus          `json:"paymentStatus" binding:"required"`
	// Buyer information (optional)
	BuyerName     *string `json:"buyerName,omitempty"`
	BuyerContact  *string `json:"buyerContact,omitempty"`
	BuyerLocation *string `json:"buyerLocation,omitempty"`
}

type UpdateSaleRequest struct {
	ProductID         *uint                  `json:"productId,omitempty"`
	ProductName       *string                `json:"productName,omitempty"`
	ProductAttributes map[string]interface{} `json:"productAttributes,omitempty"`
	Quantity          *int                   `json:"quantity,omitempty"`
	UnitPrice         *float64               `json:"unitPrice,omitempty"`
	ExtraCosts        *float64               `json:"extraCosts,omitempty"` // Optional additional costs like delivery charges
	TotalPrice        *float64               `json:"totalPrice,omitempty"`
	Currency          *string                `json:"currency,omitempty"`
	SellerID          *uint                  `json:"sellerId,omitempty"`
	PaymentStatus     *PaymentStatus         `json:"paymentStatus,omitempty"`
	// Buyer information (optional)
	BuyerName     *string `json:"buyerName,omitempty"`
	BuyerContact  *string `json:"buyerContact,omitempty"`
	BuyerLocation *string `json:"buyerLocation,omitempty"`
}

type SaleFilter struct {
	UserID    *string
	Branch    *string
	StartDate *time.Time
	EndDate   *time.Time
}
