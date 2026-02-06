package Expense

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	expenses := rg.Group("/expenses")
	{
		expenses.GET("", getAllExpensesHandler)
		expenses.GET("/:id", getExpenseHandler)
		expenses.GET("/user/:userId", getExpensesByUserHandler)
		expenses.GET("/branch/:branchId", getExpensesByBranchHandler)
		expenses.GET("/date-range", getExpensesByDateRangeHandler)
		expenses.POST("", createExpenseHandler)
		expenses.PUT("/:id", updateExpenseHandler)
		expenses.DELETE("/:id", deleteExpenseHandler)
	}
}

func getAllExpensesHandler(c *gin.Context) {
	expenses, err := GetExpenseService().GetAllExpenses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses})
}

func getExpenseHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}
	expense, err := GetExpenseService().GetExpenseByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, expense)
}

func getExpensesByUserHandler(c *gin.Context) {
	userIDParam := c.Param("userId")
	userID, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	expenses, err := GetExpenseService().GetExpensesByUser(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses})
}

func getExpensesByBranchHandler(c *gin.Context) {
	branchIDParam := c.Param("branchId")
	branchID, err := strconv.ParseUint(branchIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}
	expenses, err := GetExpenseService().GetExpensesByBranch(uint(branchID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses})
}

func getExpensesByDateRangeHandler(c *gin.Context) {
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

	expenses, err := GetExpenseService().GetExpensesByDateRange(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses})
}

func createExpenseHandler(c *gin.Context) {
	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	req.UserID = userIDUint

	// Get user's branch ID
	user, err := User.GetUserService().GetUserByID(userIDUint)
	if err != nil || user == nil || user.BranchID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user must have a branch"})
		return
	}
	req.BranchID = *user.BranchID

	expense, err := GetExpenseService().CreateExpense(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, expense)
}

func updateExpenseHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	var req UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expense, err := GetExpenseService().UpdateExpense(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, expense)
}

func deleteExpenseHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}
	if err := GetExpenseService().DeleteExpense(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "expense deleted successfully"})
}
