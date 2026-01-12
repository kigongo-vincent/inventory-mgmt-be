package Sale

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	Notification "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Notification"
	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
)

// saleService is initialized via InitializeService in main.go
func GetSaleService() *SaleService {
	return saleService
}

func RegisterRoutes(rg *gin.RouterGroup) {
	sales := rg.Group("/sales")
	{
		sales.GET("", getAllSalesHandler)
		sales.GET("/:id", getSaleHandler)
		sales.GET("/user/:userId", getSalesByUserHandler)
		sales.GET("/branch/:branch", getSalesByBranchHandler)
		sales.GET("/date-range", getSalesByDateRangeHandler)
		sales.GET("/events", salesEventsHandler) // SSE endpoint
		sales.POST("", createSaleHandler)
		sales.PUT("/:id", updateSaleHandler)
		sales.DELETE("/:id", deleteSaleHandler)
	}
}

func getAllSalesHandler(c *gin.Context) {
	sales, err := GetSaleService().GetAllSales()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales": sales})
}

func getSaleHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sale id"})
		return
	}
	sale, err := GetSaleService().GetSaleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sale)
}

func getSalesByUserHandler(c *gin.Context) {
	userIDParam := c.Param("userId")
	userID, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	sales, err := GetSaleService().GetSalesByUser(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales": sales})
}

func getSalesByBranchHandler(c *gin.Context) {
	branchParam := c.Param("branch")
	branchID, err := strconv.ParseUint(branchParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}
	sales, err := GetSaleService().GetSalesByBranch(uint(branchID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales": sales})
}

func getSalesByDateRangeHandler(c *gin.Context) {
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate format"})
		return
	}

	sales, err := GetSaleService().GetSalesByDateRange(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sales": sales})
}

func createSaleHandler(c *gin.Context) {
	var req CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get seller ID from token context (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user information not found"})
		return
	}

	sellerID, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user information"})
		return
	}

	// Set seller ID from context (override any value sent in request for security)
	req.SellerID = sellerID

	sale, err := GetSaleService().CreateSale(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create notification for sale
	notificationService := Notification.GetNotificationService()
	if notificationService != nil {
		saleID := sale.ID
		_, _ = notificationService.CreateNotification(Notification.CreateNotificationRequest{
			UserID:    sellerID,
			Type:      Notification.NotificationTypeSale,
			Title:     "Sale Recorded",
			Message:   fmt.Sprintf("Recorded sale of %d units of %s for %s %.2f", sale.Quantity, sale.ProductName, sale.Currency, sale.TotalPrice),
			RelatedID: &saleID,
		})
	}

	// Broadcast SSE event to super admins of the company
	sseService := GetSSEService()
	companyID, err := GetSaleService().GetCompanyIDFromSale(sale)
	if err == nil && companyID != nil {
		seller, _ := User.GetUserService().GetUserByID(sellerID)
		sellerName := ""
		branchName := ""
		if seller != nil {
			sellerName = seller.Name
			// Branch name is already populated in seller object
			branchName = seller.Branch
		}

		// If sale has branch populated, use that instead
		if sale.Branch != nil && sale.Branch.Name != "" {
			branchName = sale.Branch.Name
		}

		saleEvent := SaleEvent{
			Type:        "new_sale",
			SaleID:      sale.ID,
			ProductName: sale.ProductName,
			Quantity:    sale.Quantity,
			TotalPrice:  sale.TotalPrice,
			Currency:    sale.Currency,
			SellerName:  sellerName,
			BranchName:  branchName,
			CreatedAt:   sale.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		sseService.BroadcastSaleEvent(*companyID, saleEvent)
	}

	c.JSON(http.StatusCreated, sale)
}

func updateSaleHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sale id"})
		return
	}
	var req UpdateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get sale before update to check if it's a reorder
	oldSale, _ := GetSaleService().GetSaleByID(uint(id))

	sale, err := GetSaleService().UpdateSale(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create notification for sale reorder/update
	userID, exists := c.Get("user_id")
	if exists {
		if userIDUint, ok := userID.(uint); ok {
			notificationService := Notification.GetNotificationService()
			if notificationService != nil {
				saleID := sale.ID
				// Check if quantity or price changed (reorder scenario)
				isReorder := false
				if oldSale != nil {
					if (req.Quantity != nil && *req.Quantity != oldSale.Quantity) ||
						(req.TotalPrice != nil && *req.TotalPrice != oldSale.TotalPrice) {
						isReorder = true
					}
				}

				title := "Sale Updated"
				message := fmt.Sprintf("Updated sale of %s", sale.ProductName)
				if isReorder {
					title = "Sale Reordered"
					message = fmt.Sprintf("Reordered sale of %d units of %s for %s %.2f", sale.Quantity, sale.ProductName, sale.Currency, sale.TotalPrice)
				}

				_, _ = notificationService.CreateNotification(Notification.CreateNotificationRequest{
					UserID:    userIDUint,
					Type:      Notification.NotificationTypeSale,
					Title:     title,
					Message:   message,
					RelatedID: &saleID,
				})
			}
		}
	}

	c.JSON(http.StatusOK, sale)
}

func deleteSaleHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sale id"})
		return
	}
	if err := GetSaleService().DeleteSale(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sale deleted successfully"})
}

// salesEventsHandler handles SSE connections for sale events
func salesEventsHandler(c *gin.Context) {
	// Get user information from context (set by AuthMiddleware)
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

	// Get company ID from context
	companyID, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "company information not found"})
		return
	}

	companyIDPtr, ok := companyID.(*uint)
	if !ok || companyIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid company information"})
		return
	}

	// Get user role
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "role information not found"})
		return
	}

	userRole, ok := role.(User.UserRole)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid role information"})
		return
	}

	// Only allow super admins to connect
	if userRole != User.SuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "only super admins can subscribe to sale events"})
		return
	}

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Register client
	sseService := GetSSEService()
	client := sseService.RegisterClient(userIDUint, *companyIDPtr, userRole)

	// Send initial connection message
	fmt.Fprintf(c.Writer, "data: {\"type\":\"connected\"}\n\n")
	c.Writer.Flush()

	// Handle client disconnect
	notify := c.Writer.CloseNotify()
	go func() {
		<-notify
		sseService.UnregisterClient(userIDUint, *companyIDPtr)
	}()

	// Stream events to client
	for {
		select {
		case message := <-client.Channel:
			if _, err := c.Writer.Write(message); err != nil {
				// Client disconnected
				sseService.UnregisterClient(userIDUint, *companyIDPtr)
				return
			}
			c.Writer.Flush()
		case <-notify:
			// Client disconnected
			sseService.UnregisterClient(userIDUint, *companyIDPtr)
			return
		}
	}
}
