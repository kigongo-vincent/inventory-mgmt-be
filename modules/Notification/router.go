package Notification

import (
	"net/http"
	"strconv"

	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	notifications := rg.Group("/notifications")
	{
		// All notification routes require authentication
		notifications.Use(User.AuthMiddleware())
		{
			notifications.GET("", getNotificationsHandler)
			notifications.GET("/unread", getUnreadNotificationsHandler)
			notifications.GET("/unread/count", getUnreadCountHandler)
			notifications.GET("/:id", getNotificationHandler)
			notifications.POST("", createNotificationHandler)
			notifications.PUT("/:id/read", markAsReadHandler)
			notifications.PUT("/read-all", markAllAsReadHandler)
			notifications.DELETE("/:id", deleteNotificationHandler)
			notifications.DELETE("", deleteAllNotificationsHandler)
		}
	}
}

func getNotificationsHandler(c *gin.Context) {
	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	notifications, err := GetNotificationService().GetNotificationsByUser(userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func getUnreadNotificationsHandler(c *gin.Context) {
	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	notifications, err := GetNotificationService().GetUnreadNotificationsByUser(userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func getUnreadCountHandler(c *gin.Context) {
	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	count, err := GetNotificationService().GetUnreadCount(userIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func getNotificationHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	notification, err := GetNotificationService().GetNotificationByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Verify the notification belongs to the user
	if notification.UserID != userIDUint {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func createNotificationHandler(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	// Users can only create notifications for themselves (unless they're super_admin)
	user, err := User.GetUserService().GetUserByID(userIDUint)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Only allow users to create notifications for themselves, or super_admin to create for any user
	if user.Role != User.SuperAdmin && req.UserID != userIDUint {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only create notifications for yourself"})
		return
	}

	notification, err := GetNotificationService().CreateNotification(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If notification is nil, it means it was filtered by preferences
	if notification == nil {
		c.JSON(http.StatusOK, gin.H{"message": "notification filtered by user preferences"})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func markAsReadHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	if err := GetNotificationService().MarkAsRead(uint(id), userIDUint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

func markAllAsReadHandler(c *gin.Context) {
	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	if err := GetNotificationService().MarkAllAsRead(userIDUint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

func deleteNotificationHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	if err := GetNotificationService().DeleteNotification(uint(id), userIDUint); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification deleted successfully"})
}

func deleteAllNotificationsHandler(c *gin.Context) {
	// Get user ID from JWT token (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	if err := GetNotificationService().DeleteAllNotifications(userIDUint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications deleted successfully"})
}
