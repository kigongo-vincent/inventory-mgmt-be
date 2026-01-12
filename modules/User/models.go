package User

import (
	"gorm.io/gorm"
)

type UserRole string

const (
	SuperAdmin UserRole = "super_admin"
	User       UserRole = "user"
)

type SyncStatus string

const (
	Online  SyncStatus = "online"
	Offline SyncStatus = "offline"
	Synced  SyncStatus = "synced"
)

type UserModel struct {
	gorm.Model
	Name              string      `json:"name" gorm:"not null"`
	Username          string      `json:"username" gorm:"uniqueIndex;not null"`
	Password          string      `json:"password" gorm:"not null"` // In production, this should be hashed
	Role              UserRole    `json:"role" gorm:"not null"`
	Email             *string     `json:"email,omitempty"`
	Phone             *string     `json:"phone,omitempty"`
	ProfilePictureURI *string     `json:"profilePictureUri,omitempty"`
	SyncStatus        *SyncStatus `json:"syncStatus,omitempty"`

	// Foreign Keys
	BranchID *uint `json:"branchId,omitempty" gorm:"index;not null"` // All users must belong to a branch

	// Computed fields for JSON response (not stored in DB)
	Branch    string `json:"branch,omitempty" gorm:"-"`    // Branch name - computed from BranchID
	Company   string `json:"company,omitempty" gorm:"-"`    // Company name - computed from BranchID via Branch
	CompanyID *uint  `json:"companyId,omitempty" gorm:"-"`  // Company ID - computed from BranchID via Branch
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User  UserModel `json:"user"`
	Token string    `json:"token"` // JWT token for authentication
}

type CreateUserRequest struct {
	Name              string   `json:"name" binding:"required"`
	Username          string   `json:"username" binding:"required"`
	Password          string   `json:"password" binding:"required"`
	Role              UserRole `json:"role" binding:"required"`
	BranchID          *uint    `json:"branchId" binding:"required"` // All users must belong to a branch
	Email             *string  `json:"email,omitempty"`
	Phone             *string  `json:"phone,omitempty"`
	ProfilePictureURI *string  `json:"profilePictureUri,omitempty"`
}

type UpdateUserRequest struct {
	Name              *string   `json:"name,omitempty"`
	Username          *string   `json:"username,omitempty"`
	Password          *string   `json:"password,omitempty"`
	Role              *UserRole `json:"role,omitempty"`
	BranchID          *uint     `json:"branchId,omitempty"`
	Email             *string   `json:"email,omitempty"`
	Phone             *string   `json:"phone,omitempty"`
	ProfilePictureURI *string   `json:"profilePictureUri,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// NotificationPreferences model - one-to-one relationship with User
type NotificationPreferences struct {
	gorm.Model
	UserID              uint  `json:"userId" gorm:"uniqueIndex;not null"` // Foreign key to User
	SalesNotifications  bool  `json:"salesNotifications" gorm:"default:true"`
	InventoryAlerts     bool  `json:"inventoryAlerts" gorm:"default:true"`
	UserActivity        bool  `json:"userActivity" gorm:"default:false"`
	SystemUpdates       bool  `json:"systemUpdates" gorm:"default:true"`
	EmailNotifications  bool  `json:"emailNotifications" gorm:"default:false"`
	PushNotifications   bool  `json:"pushNotifications" gorm:"default:true"`
}

type UpdateNotificationPreferencesRequest struct {
	SalesNotifications  *bool `json:"salesNotifications,omitempty"`
	InventoryAlerts     *bool `json:"inventoryAlerts,omitempty"`
	UserActivity        *bool `json:"userActivity,omitempty"`
	SystemUpdates       *bool `json:"systemUpdates,omitempty"`
	EmailNotifications  *bool `json:"emailNotifications,omitempty"`
	PushNotifications   *bool `json:"pushNotifications,omitempty"`
}

// Branch models (moved from Branch package to avoid import cycle)
type Branch struct {
	gorm.Model
	CompanyID   uint    `json:"companyId" gorm:"not null;index"`
	AdminUserID *uint   `json:"adminUserId,omitempty" gorm:"index"` // Foreign key to User (company admin)
	Name        string  `json:"name" gorm:"not null"`
	Address     *string `json:"address,omitempty"`
	Phone       *string `json:"phone,omitempty"`
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
