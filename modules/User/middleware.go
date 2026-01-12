package User

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT tokens and fetches user info from database
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
				"code":  "MISSING_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
				"code":  "INVALID_AUTH_FORMAT",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "token required",
				"code":  "MISSING_TOKEN",
			})
			c.Abort()
			return
		}

		// Validate JWT token
		claims, err := ValidateJWT(tokenString)
		if err != nil {
			// Check if token is expired
			var errorCode string
			if err.Error() == "token is expired" || strings.Contains(err.Error(), "expired") {
				errorCode = "TOKEN_EXPIRED"
			} else {
				errorCode = "INVALID_TOKEN"
			}
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
				"code":  errorCode,
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Convert UserID from string to uint
		userID, err := strconv.ParseUint(claims.UserID, 10, 32)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user id in token",
				"code":  "INVALID_USER_ID",
			})
			c.Abort()
			return
		}

		// Fetch full user information from database
		user, err := GetUserService().GetUserByID(uint(userID))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user not found",
				"code":  "USER_NOT_FOUND",
			})
			c.Abort()
			return
		}

		// Determine company ID from user's branch (all users belong to a branch)
		var companyID *uint
		if user.BranchID != nil {
			// Get company ID from user's branch
			companyIDFromBranch, err := GetUserService().GetCompanyIDFromBranch(*user.BranchID)
			if err == nil && companyIDFromBranch != nil {
				companyID = companyIDFromBranch
			}
		}

		// Store user information in context (locals) for use in handlers
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Set("branch_id", user.BranchID)
		c.Set("branch", user.Branch) // Branch name (computed)
		c.Set("company_id", companyID) // Company ID (from user or branch)
		c.Set("company", user.Company) // Company name (computed)
		c.Set("email", user.Email)
		c.Set("phone", user.Phone)
		c.Set("token", tokenString)
		
		// Store full user object for convenience
		c.Set("user", user)

		c.Next()
	}
}

// AdminMiddleware checks if user has admin role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userRole, ok := role.(UserRole)
		if !ok || userRole != SuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
