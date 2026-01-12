package Notification

import (
	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationTypeSale      NotificationType = "sale"
	NotificationTypeInventory NotificationType = "inventory"
	NotificationTypeUser      NotificationType = "user"
	NotificationTypeSystem    NotificationType = "system"
)

type Notification struct {
	gorm.Model
	UserID    uint             `json:"userId" gorm:"not null;index"`
	Type      NotificationType `json:"type" gorm:"not null"`
	Title     string           `json:"title" gorm:"not null"`
	Message   string           `json:"message" gorm:"not null"`
	Read      bool             `json:"read" gorm:"default:false"`
	RelatedID *uint            `json:"relatedId,omitempty"` // ID of related sale, product, user, etc.
}

type CreateNotificationRequest struct {
	UserID    uint             `json:"userId" binding:"required"`
	Type      NotificationType `json:"type" binding:"required"`
	Title     string           `json:"title" binding:"required"`
	Message   string           `json:"message" binding:"required"`
	RelatedID *uint            `json:"relatedId,omitempty"`
}

type UpdateNotificationRequest struct {
	Read *bool `json:"read,omitempty"`
}
