package Branch

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
	// "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	branches := rg.Group("/branches")
	{
		// All branch routes require authentication
		branches.Use(User.AuthMiddleware())
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
