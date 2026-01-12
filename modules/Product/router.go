package Product

import (
	"fmt"
	"net/http"
	"strconv"

	Notification "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Notification"
	"github.com/gin-gonic/gin"
)

// productService is initialized via InitializeService in main.go
func GetProductService() *ProductService {
	return productService
}

func RegisterRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")
	{
		products.GET("", getAllProductsHandler) // Returns products for user's company
		products.GET("/:id", getProductHandler)
		products.POST("", createProductHandler) // Company ID from middleware
		products.PUT("/:id", updateProductHandler)
		products.DELETE("/:id", deleteProductHandler)
		products.POST("/:id/reduce", reduceProductQuantityHandler)
	}
}

func getAllProductsHandler(c *gin.Context) {
	// Get company ID from middleware context
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

	products, err := GetProductService().GetAllProductsByCompany(*companyIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

func getProductHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	product, err := GetProductService().GetProductByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Verify product belongs to user's company
	companyID, exists := c.Get("company_id")
	if exists {
		companyIDPtr, ok := companyID.(*uint)
		if ok && companyIDPtr != nil {
			if product.CompanyID != *companyIDPtr {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, product)
}


func createProductHandler(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get company ID from middleware context (set by AuthMiddleware)
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

	// Override company ID from request with the one from middleware for security
	req.CompanyID = *companyIDPtr

	product, err := GetProductService().CreateProduct(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create notification for stock addition
	userID, exists := c.Get("user_id")
	if exists {
		if userIDUint, ok := userID.(uint); ok {
			notificationService := Notification.GetNotificationService()
			if notificationService != nil {
				productID := product.ID
				_, _ = notificationService.CreateNotification(Notification.CreateNotificationRequest{
					UserID:    userIDUint,
					Type:      Notification.NotificationTypeInventory,
					Title:     "Stock Added",
					Message:   fmt.Sprintf("Added %d units of %s to inventory", product.Quantity, product.Name),
					RelatedID: &productID,
				})
			}
		}
	}

	c.JSON(http.StatusCreated, product)
}

func updateProductHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get company ID from middleware and verify product belongs to user's company
	companyID, exists := c.Get("company_id")
	if exists {
		companyIDPtr, ok := companyID.(*uint)
		if ok && companyIDPtr != nil {
			// Verify product belongs to user's company
			product, err := GetProductService().GetProductByID(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			if product.CompanyID != *companyIDPtr {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
			// Prevent changing company ID
			req.CompanyID = nil
		}
	}

	// Get product before update to check quantity change
	var oldQuantity int
	oldProduct, _ := GetProductService().GetProductByID(uint(id))
	if oldProduct != nil {
		oldQuantity = oldProduct.Quantity
	}

	product, err := GetProductService().UpdateProduct(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create notification if quantity increased (stock reorder)
	if req.Quantity != nil && *req.Quantity > oldQuantity {
		userID, exists := c.Get("user_id")
		if exists {
			if userIDUint, ok := userID.(uint); ok {
				notificationService := Notification.GetNotificationService()
				if notificationService != nil {
					productID := product.ID
					quantityAdded := *req.Quantity - oldQuantity
					_, _ = notificationService.CreateNotification(Notification.CreateNotificationRequest{
						UserID:    userIDUint,
						Type:      Notification.NotificationTypeInventory,
						Title:     "Stock Reordered",
						Message:   fmt.Sprintf("Reordered %d units of %s. New total: %d units", quantityAdded, product.Name, product.Quantity),
						RelatedID: &productID,
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, product)
}

func deleteProductHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	
	// Verify product belongs to user's company
	companyID, exists := c.Get("company_id")
	if exists {
		companyIDPtr, ok := companyID.(*uint)
		if ok && companyIDPtr != nil {
			product, err := GetProductService().GetProductByID(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			if product.CompanyID != *companyIDPtr {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}
	}

	if err := GetProductService().DeleteProduct(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product deleted successfully"})
}

func reduceProductQuantityHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	quantityStr := c.Query("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quantity"})
		return
	}

	// Verify product belongs to user's company
	companyID, exists := c.Get("company_id")
	if exists {
		companyIDPtr, ok := companyID.(*uint)
		if ok && companyIDPtr != nil {
			product, err := GetProductService().GetProductByID(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			if product.CompanyID != *companyIDPtr {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}
	}

	if err := GetProductService().ReduceProductQuantity(uint(id), quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, _ := GetProductService().GetProductByID(uint(id))
	c.JSON(http.StatusOK, product)
}
