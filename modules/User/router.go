package User

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetUserService returns the initialized user service
func GetUserService() *UserService {
	return userService
}

func RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		// Public routes
		users.POST("/login", loginHandler)

		// Protected routes
		protected := users.Group("")
		protected.Use(AuthMiddleware())
		{
			protected.GET("", getAllUsersHandler)
			protected.GET("/:id", getUserHandler)
			protected.GET("/branch/:branchId", getUsersByBranchHandler)
			protected.POST("", createUserHandler)
			protected.PUT("/:id", updateUserHandler)
			protected.POST("/:id/change-password", changePasswordHandler)
			protected.DELETE("/:id", deleteUserHandler)
			// Notification preferences routes
			protected.GET("/:id/notification-preferences", getNotificationPreferencesHandler)
			protected.PUT("/:id/notification-preferences", updateNotificationPreferencesHandler)
		}
	}
}

func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetUserService().Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	response := LoginResponse{
		User:  *user,
		Token: token,
	}

	c.JSON(http.StatusOK, response)
}

func getAllUsersHandler(c *gin.Context) {
	users, err := GetUserService().GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func getUserHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	user, err := GetUserService().GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func getUsersByBranchHandler(c *gin.Context) {
	branchIdParam := c.Param("branchId")
	branchId, err := strconv.ParseUint(branchIdParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}
	users, err := GetUserService().GetUsersByBranch(uint(branchId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func createUserHandler(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetUserService().CreateUser(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func updateUserHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetUserService().UpdateUser(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func changePasswordHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
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

	// Users can only change their own password (unless they're super_admin)
	// Get user to check role
	user, err := GetUserService().GetUserByID(userIDUint)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Only allow users to change their own password, or super_admin to change any password
	if user.Role != SuperAdmin && uint(id) != userIDUint {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only change your own password"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := GetUserService().ChangePassword(uint(id), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

func deleteUserHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if err := GetUserService().DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// Branch Routes (moved from Branch package to avoid import cycle)
func RegisterBranchRoutes(rg *gin.RouterGroup) {
	branches := rg.Group("/branches")
	{
		// All branch routes require authentication
		branches.Use(AuthMiddleware())
		{
			branches.GET("", getAllBranchesHandler)
			branches.GET("/:id", getBranchHandler)
			branches.GET("/company/:companyId", getBranchesByCompanyHandler)
			branches.POST("", createBranchHandler)
			branches.PUT("/:id", updateBranchHandler)
			branches.DELETE("/:id", deleteBranchHandler)
		}
	}
}

func getAllBranchesHandler(c *gin.Context) {
	branches, err := GetBranchService().GetAllBranches()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

func getBranchHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}
	branch, err := GetBranchService().GetBranchByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, branch)
}

func getBranchesByCompanyHandler(c *gin.Context) {
	// Get company ID from middleware context (set by AuthMiddleware) for security
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

	branches, err := GetBranchService().GetBranchesByCompany(*companyIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

func createBranchHandler(c *gin.Context) {
	var req CreateBranchRequest
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

	branch, err := GetBranchService().CreateBranch(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, branch)
}

func updateBranchHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}
	var req UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get company ID from middleware and verify branch belongs to user's company
	companyID, exists := c.Get("company_id")
	if exists {
		companyIDPtr, ok := companyID.(*uint)
		if ok && companyIDPtr != nil {
			// Verify branch belongs to user's company
			branch, err := GetBranchService().GetBranchByID(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
				return
			}
			if branch.CompanyID != *companyIDPtr {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
			// Prevent changing company ID
			req.CompanyID = nil
		}
	}

	branch, err := GetBranchService().UpdateBranch(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, branch)
}

func deleteBranchHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid branch id"})
		return
	}
	
	// Verify branch belongs to user's company
	companyID, exists := c.Get("company_id")
	if exists {
		companyIDPtr, ok := companyID.(*uint)
		if ok && companyIDPtr != nil {
			branch, err := GetBranchService().GetBranchByID(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "branch not found"})
				return
			}
			if branch.CompanyID != *companyIDPtr {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		}
	}

	if err := GetBranchService().DeleteBranch(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "branch deleted successfully"})
}

func getNotificationPreferencesHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
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

	// Users can only access their own notification preferences (unless they're super_admin)
	user, err := GetUserService().GetUserByID(userIDUint)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Only allow users to access their own preferences, or super_admin to access any
	if user.Role != SuperAdmin && uint(id) != userIDUint {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only access your own notification preferences"})
		return
	}

	prefs, err := GetUserService().GetNotificationPreferences(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func updateNotificationPreferencesHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
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

	// Users can only update their own notification preferences (unless they're super_admin)
	user, err := GetUserService().GetUserByID(userIDUint)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Only allow users to update their own preferences, or super_admin to update any
	if user.Role != SuperAdmin && uint(id) != userIDUint {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only update your own notification preferences"})
		return
	}

	var req UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs, err := GetUserService().UpdateNotificationPreferences(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
