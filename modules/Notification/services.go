package Notification

import (
	"errors"
	"fmt"

	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
	"gorm.io/gorm"
)

var notificationService *NotificationService

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// InitializeService initializes the notification service with a database connection
func InitializeService(db *gorm.DB) {
	notificationService = &NotificationService{db: db}
}

// GetNotificationService returns the initialized notification service
func GetNotificationService() *NotificationService {
	return notificationService
}

// CreateNotification creates a new notification for a user
// It checks the user's notification preferences before creating
func (s *NotificationService) CreateNotification(req CreateNotificationRequest) (*Notification, error) {
	// Get user's notification preferences
	userService := User.GetUserService()
	if userService == nil {
		return nil, errors.New("user service not initialized")
	}

	prefs, err := userService.GetNotificationPreferences(req.UserID)
	if err != nil {
		// If preferences don't exist, create with defaults and continue
		prefs, err = userService.GetNotificationPreferences(req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get notification preferences: %w", err)
		}
	}

	// Check if user has enabled notifications for this type
	shouldNotify := false
	switch req.Type {
	case NotificationTypeSale:
		shouldNotify = prefs.SalesNotifications
	case NotificationTypeInventory:
		shouldNotify = prefs.InventoryAlerts
	case NotificationTypeUser:
		shouldNotify = prefs.UserActivity
	case NotificationTypeSystem:
		shouldNotify = prefs.SystemUpdates
	}

	// If user has disabled this type of notification, don't create it
	if !shouldNotify {
		return nil, nil // Return nil without error - notification was filtered by preferences
	}

	notification := &Notification{
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Read:      false,
		RelatedID: req.RelatedID,
	}

	if err := s.db.Create(notification).Error; err != nil {
		return nil, err
	}

	return notification, nil
}

// CreateNotificationForUsers creates notifications for multiple users
// Useful for system-wide notifications or notifications for all users in a branch/company
func (s *NotificationService) CreateNotificationForUsers(userIDs []uint, req CreateNotificationRequest) ([]*Notification, error) {
	var notifications []*Notification
	for _, userID := range userIDs {
		req.UserID = userID
		notification, err := s.CreateNotification(req)
		if err != nil {
			// Log error but continue with other users
			fmt.Printf("Failed to create notification for user %d: %v\n", userID, err)
			continue
		}
		if notification != nil { // Only add if notification was created (not filtered by preferences)
			notifications = append(notifications, notification)
		}
	}
	return notifications, nil
}

// GetNotificationsByUser retrieves all notifications for a user
func (s *NotificationService) GetNotificationsByUser(userID uint) ([]*Notification, error) {
	var notifications []*Notification
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetUnreadNotificationsByUser retrieves unread notifications for a user
func (s *NotificationService) GetUnreadNotificationsByUser(userID uint) ([]*Notification, error) {
	var notifications []*Notification
	if err := s.db.Where("user_id = ? AND read = ?", userID, false).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetNotificationByID retrieves a notification by ID
func (s *NotificationService) GetNotificationByID(id uint) (*Notification, error) {
	var notification Notification
	if err := s.db.First(&notification, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("notification not found")
		}
		return nil, err
	}
	return &notification, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(id uint, userID uint) error {
	var notification Notification
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("notification not found")
		}
		return err
	}

	notification.Read = true
	if err := s.db.Save(&notification).Error; err != nil {
		return err
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(userID uint) error {
	if err := s.db.Model(&Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Update("read", true).Error; err != nil {
		return err
	}
	return nil
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(id uint, userID uint) error {
	var notification Notification
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("notification not found")
		}
		return err
	}

	if err := s.db.Delete(&notification).Error; err != nil {
		return err
	}

	return nil
}

// DeleteAllNotifications deletes all notifications for a user
func (s *NotificationService) DeleteAllNotifications(userID uint) error {
	if err := s.db.Where("user_id = ?", userID).Delete(&Notification{}).Error; err != nil {
		return err
	}
	return nil
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	if err := s.db.Model(&Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
