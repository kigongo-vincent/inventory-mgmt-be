package User

import (
	"errors"
	"fmt"
	"os"

	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Company"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var userService *UserService

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{}
}

// InitializeService initializes the user service and branch service with a database connection
func InitializeService(db *gorm.DB) {
	userService = &UserService{db: db}
	// Initialize branch service (moved to User package to avoid import cycle)
	InitializeBranchService(db)
	// Initialize company admin on startup
	if err := userService.InitializeCompanyAdmin(); err != nil {
		fmt.Printf("Warning: Failed to initialize company admin: %v\n", err)
	}
	// Initialize mock data for development
	userService.InitializeMockData()
}

func (s *UserService) Login(username, password string) (*UserModel, error) {
	var user UserModel
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Compare hashed password with provided password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	s.populateBranchAndCompany(&user)
	return &user, nil
}

func (s *UserService) GetUserByID(id uint) (*UserModel, error) {
	var user UserModel
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	s.populateBranchAndCompany(&user)
	return &user, nil
}

func (s *UserService) GetUserByUsername(username string) (*UserModel, error) {
	var user UserModel
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	s.populateBranchAndCompany(&user)
	return &user, nil
}

func (s *UserService) GetAllUsers() ([]*UserModel, error) {
	var users []*UserModel
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		s.populateBranchAndCompany(users[i])
	}
	return users, nil
}

func (s *UserService) GetUsersByBranch(branchID uint) ([]*UserModel, error) {
	var users []*UserModel
	if err := s.db.Where("branch_id = ?", branchID).Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		s.populateBranchAndCompany(users[i])
	}
	return users, nil
}

func (s *UserService) CreateUser(req CreateUserRequest) (*UserModel, error) {
	// Check if username already exists
	var existingUser UserModel
	if err := s.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, errors.New("username already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// All users must have a BranchID (company is derived from branch)
	if req.BranchID == nil {
		return nil, errors.New("branchId is required for all users")
	}

	// Verify branch exists
	var branch Branch
	if err := s.db.First(&branch, "id = ?", *req.BranchID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	// Hash the password before storing
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &UserModel{
		Name:              req.Name,
		Username:          req.Username,
		Password:          hashedPassword,
		Role:              req.Role,
		BranchID:          req.BranchID,
		Email:             req.Email,
		Phone:             req.Phone,
		ProfilePictureURI: req.ProfilePictureURI,
		SyncStatus:        nil, // Default to nil (will be set to "online" for new users)
	}

	fmt.Printf("[CreateUser] Before DB create - BranchID: %v (type: %T)\n", user.BranchID, user.BranchID)
	if req.BranchID != nil {
		fmt.Printf("[CreateUser] BranchID from request: %v (type: %T)\n", *req.BranchID, *req.BranchID)
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	fmt.Printf("[CreateUser] After DB create - ID: %v, BranchID: %v\n", user.ID, user.BranchID)

	// If BranchID was provided but is NULL after create, force update it immediately
	if req.BranchID != nil && (user.BranchID == nil || (user.BranchID != nil && *user.BranchID != *req.BranchID)) {
		fmt.Printf("[CreateUser] WARNING: BranchID mismatch! Requested: %v, Got: %v, forcing update...\n", *req.BranchID, user.BranchID)
		// Use direct SQL update to ensure it works
		if err := s.db.Exec("UPDATE user_models SET branch_id = ? WHERE id = ?", *req.BranchID, user.ID).Error; err != nil {
			fmt.Printf("[CreateUser] ERROR: Direct SQL update failed: %v\n", err)
			// Fallback to GORM Update
			if err := s.db.Model(user).Update("branch_id", *req.BranchID).Error; err != nil {
				fmt.Printf("[CreateUser] ERROR: GORM Update also failed: %v\n", err)
			}
		}
		// Re-fetch to get updated value
		if err := s.db.First(user, user.ID).Error; err == nil {
			fmt.Printf("[CreateUser] After force update - BranchID: %v\n", user.BranchID)
		}
	}

	s.populateBranchAndCompany(user)
	return user, nil
}

func (s *UserService) UpdateUser(id uint, req UpdateUserRequest) (*UserModel, error) {
	var user UserModel
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// Check username uniqueness if username is being updated
	if req.Username != nil && *req.Username != user.Username {
		var existingUser UserModel
		if err := s.db.Where("username = ? AND id != ?", *req.Username, id).First(&existingUser).Error; err == nil {
			return nil, errors.New("username already exists")
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		user.Username = *req.Username
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Password != nil {
		// Hash the new password before storing
		hashedPassword, err := hashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.Password = hashedPassword
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.BranchID != nil {
		// Verify branch exists
		var branch Branch
		if err := s.db.First(&branch, "id = ?", *req.BranchID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.New("branch not found")
			}
			return nil, err
		}
		user.BranchID = req.BranchID
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.ProfilePictureURI != nil {
		user.ProfilePictureURI = req.ProfilePictureURI
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	s.populateBranchAndCompany(&user)
	return &user, nil
}

func (s *UserService) ChangePassword(id uint, req ChangePasswordRequest) error {
	var user UserModel
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Check if new password is same as old password
	if req.OldPassword == req.NewPassword {
		return errors.New("new password must be different from current password")
	}

	// Check minimum length
	if len(req.NewPassword) < 6 {
		return errors.New("new password must be at least 6 characters long")
	}

	// Hash the new password
	hashedPassword, err := hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.Password = hashedPassword
	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

func (s *UserService) DeleteUser(id uint) error {
	var user UserModel
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return err
	}

	if err := s.db.Delete(&user).Error; err != nil {
		return err
	}

	return nil
}

// InitializeCompanyAdmin creates the company, main branch, and owner from environment variables
// Order: 1. Create Company 2. Create Main Branch 3. Create Owner (admin user) with both company and branch IDs
// This is called on backend startup to ensure they always exist
func (s *UserService) InitializeCompanyAdmin() error {
	// Get environment variables
	companyName := os.Getenv("COMPANY_NAME")
	companyEmail := os.Getenv("COMPANY_EMAIL")
	ownerName := os.Getenv("OWNER_NAME")
	ownerEmail := os.Getenv("OWNER_EMAIL")
	ownerPassword := os.Getenv("OWNER_PASSWORD")
	ownerUsername := os.Getenv("OWNER_USERNAME")
	branchName := os.Getenv("BRANCH_NAME")

	// Set defaults if not provided
	if companyName == "" {
		companyName = "Gas Center"
	}
	if companyEmail == "" {
		companyEmail = "info@gascenter.com"
	}
	if ownerName == "" {
		ownerName = "Owner"
	}
	if ownerEmail == "" {
		ownerEmail = "owner@gascenter.com"
	}
	if ownerPassword == "" {
		ownerPassword = "admin123"
	}
	if ownerUsername == "" {
		ownerUsername = ownerEmail // Use email as username if not provided
	}
	if branchName == "" {
		branchName = "Main"
	}

	// Note: Password defaults to "admin123" if not set, but it's recommended to set OWNER_PASSWORD in production

	// Step 1: Get or create Company
	companyService := Company.GetCompanyService()
	if companyService == nil {
		return fmt.Errorf("company service not initialized")
	}

	var company *Company.Company
	// Try to find company by name first
	var companies []*Company.Company
	companies, err := companyService.GetAllCompanies()
	if err == nil {
		for _, c := range companies {
			if c.Name == companyName {
				company = c
				break
			}
		}
	}

	if company == nil {
		// Company doesn't exist, create it
		companyReq := Company.CreateCompanyRequest{
			Name:  companyName,
			Email: companyEmail,
		}
		company, err = companyService.CreateCompany(companyReq)
		if err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}
		fmt.Printf("Company created: %s (%s)\n", company.Name, company.Email)
	} else {
		fmt.Printf("Company already exists: %s (%s)\n", company.Name, company.Email)
	}

	// Step 2: Get or create Main Branch - MUST happen before user creation
	// This is CRITICAL - branch MUST exist before user is created
	fmt.Printf("[INIT] ===== STEP 2: CREATING BRANCH FIRST =====\n")
	fmt.Printf("[INIT] Company ID: %v\n", company.ID)

	branchService := GetBranchService()
	if branchService == nil {
		return fmt.Errorf("branch service not initialized")
	}

	branches, err := branchService.GetBranchesByCompany(company.ID)
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	var mainBranch *Branch
	// Check if Main branch exists
	for _, b := range branches {
		if b.Name == branchName {
			mainBranch = b
			fmt.Printf("[INIT] Found existing branch: id=%v, name=%s\n", mainBranch.ID, mainBranch.Name)
			break
		}
	}

	if mainBranch == nil {
		// Create Main branch - CRITICAL: This must complete before ANY user operations
		fmt.Printf("[INIT] Branch does not exist, creating now...\n")
		branchReq := CreateBranchRequest{
			CompanyID: company.ID,
			Name:      branchName,
		}
		fmt.Printf("[INIT] Creating branch: name=%s, companyId=%v\n", branchName, company.ID)
		mainBranch, err = branchService.CreateBranch(branchReq)
		if err != nil {
			return fmt.Errorf("failed to create main branch: %w", err)
		}
		if mainBranch == nil {
			return fmt.Errorf("branch creation returned nil")
		}
		if mainBranch.ID == 0 {
			return fmt.Errorf("branch created but ID is zero/invalid")
		}
		fmt.Printf("[INIT] ✓ Branch created successfully: id=%v, name=%s\n", mainBranch.ID, mainBranch.Name)
	}

	// CRITICAL CHECK: Verify branch exists before proceeding to user creation
	if mainBranch == nil {
		return fmt.Errorf("FATAL: mainBranch is nil after creation/retrieval - cannot proceed to user creation")
	}
	if mainBranch.ID == 0 {
		return fmt.Errorf("FATAL: mainBranch.ID is zero - cannot proceed to user creation")
	}
	fmt.Printf("[INIT] ✓ Branch verified: ID=%v, Name=%s\n", mainBranch.ID, mainBranch.Name)
	fmt.Printf("[INIT] ===== BRANCH CREATION COMPLETE - NOW PROCEEDING TO USER CREATION =====\n")

	// IMMEDIATE FIX: Update any existing users with NULL branch_id BEFORE creating new user
	// This handles the case where user was created before branch in a previous run
	fmt.Printf("[INIT] Pre-check: Fixing any existing users with NULL branch_id...\n")
	if err := s.db.Exec("UPDATE user_models SET branch_id = ? WHERE role = ? AND (branch_id IS NULL OR branch_id = '')", mainBranch.ID, SuperAdmin).Error; err != nil {
		fmt.Printf("[INIT] Warning: Pre-check update failed: %v\n", err)
	} else {
		rowsAffected := s.db.RowsAffected
		if rowsAffected > 0 {
			fmt.Printf("[INIT] Pre-check: Fixed %d existing users with NULL branch_id\n", rowsAffected)
		}
	}

	// Step 3: Get or create Owner (admin user) with both company and branch from the start
	// CRITICAL: Ensure branch exists before creating user
	if mainBranch == nil {
		return fmt.Errorf("mainBranch is nil - cannot create owner without branch")
	}
	fmt.Printf("[INIT] Step 3: Creating owner user. Branch ID: %v, Branch exists: %v\n", mainBranch.ID, mainBranch != nil)

	var owner *UserModel
	var existingOwner UserModel
	if err := s.db.Where("username = ? AND role = ?", ownerUsername, SuperAdmin).First(&existingOwner).Error; err == nil {
		// Owner already exists, update password, company, and branch if needed
		owner = &existingOwner
		needsUpdate := false

		// Check if password needs to be updated (compare with hashed version)
		if err := bcrypt.CompareHashAndPassword([]byte(owner.Password), []byte(ownerPassword)); err != nil {
			// Password doesn't match, needs to be updated
			hashedPassword, hashErr := hashPassword(ownerPassword)
			if hashErr != nil {
				return fmt.Errorf("failed to hash password: %w", hashErr)
			}
			owner.Password = hashedPassword
			needsUpdate = true
		}
		// Always ensure branch_id is set - this is critical for all users
		if owner.BranchID == nil || *owner.BranchID != mainBranch.ID {
			owner.BranchID = &mainBranch.ID
			needsUpdate = true
		}

		// Always save if branch_id was NULL or different to ensure it's set
		if needsUpdate {
			if err := s.db.Save(owner).Error; err != nil {
				return fmt.Errorf("failed to update owner: %w", err)
			}
			fmt.Printf("Owner updated: %s (companyId: %d, branchId: %d)\n",
				owner.Username, company.ID, mainBranch.ID)
		} else {
			fmt.Printf("Owner already exists: %s (companyId: %d, branchId: %d)\n",
				owner.Username, company.ID, mainBranch.ID)
		}

		// Double-check and force update branch_id if it's still NULL (safety check)
		if owner.BranchID == nil {
			owner.BranchID = &mainBranch.ID
			if err := s.db.Save(owner).Error; err != nil {
				return fmt.Errorf("failed to force update branch_id: %w", err)
			}
			fmt.Printf("Force updated branch_id for owner: %s (branchId: %d)\n", owner.Username, mainBranch.ID)
		}

		// Update branch to link it to the admin user
		if mainBranch.AdminUserID == nil || *mainBranch.AdminUserID != owner.ID {
			updateReq := UpdateBranchRequest{
				AdminUserID: &owner.ID,
			}
			_, err = branchService.UpdateBranch(mainBranch.ID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update branch with admin user ID: %w", err)
			}
			fmt.Printf("Branch %s linked to admin user: %s (userId: %d)\n", mainBranch.Name, owner.Username, owner.ID)
		}
	} else if err == gorm.ErrRecordNotFound {
		// Create the owner user with both company ID and branch ID from the start
		fmt.Printf("[INIT] Creating owner user: username=%s, companyId=%d, branchId=%d\n",
			ownerUsername, company.ID, mainBranch.ID)

		// CRITICAL: Verify branch exists and has valid ID before creating user
		if mainBranch == nil {
			return fmt.Errorf("cannot create owner: mainBranch is nil")
		}
		if mainBranch.ID == 0 {
			return fmt.Errorf("cannot create owner: mainBranch.ID is zero/invalid")
		}

		fmt.Printf("[INIT] Verified branch exists: ID=%v, Name=%s\n", mainBranch.ID, mainBranch.Name)

		emailPtr := &ownerEmail
		branchID := mainBranch.ID // Store in local variable to ensure it's captured

		req := CreateUserRequest{
			Name:     ownerName,
			Username: ownerUsername,
			Password: ownerPassword,
			Role:     SuperAdmin,
			BranchID: &branchID, // Owner belongs to main branch (company derived from branch)
			Email:    emailPtr,
		}

		fmt.Printf("[INIT] CreateUserRequest: BranchID=%v (type: %T)\n",
			req.BranchID, req.BranchID)
		if req.BranchID != nil {
			fmt.Printf("[INIT] BranchID value: %v\n", *req.BranchID)
		}

		owner, err = s.CreateUser(req)
		if err != nil {
			return fmt.Errorf("failed to create owner: %w", err)
		}

		fmt.Printf("[INIT] User created, checking values: BranchID=%v\n", owner.BranchID)

		// Re-fetch from database to ensure we have the actual stored values
		var dbUser UserModel
		if err := s.db.First(&dbUser, owner.ID).Error; err != nil {
			return fmt.Errorf("failed to re-fetch user from database: %w", err)
		}
		fmt.Printf("[INIT] User from DB: BranchID=%v\n", dbUser.BranchID)
		if dbUser.BranchID == nil || *dbUser.BranchID != mainBranch.ID {
			// Force update branch_id if it's NULL - try multiple methods
			fmt.Printf("[INIT] WARNING: branch_id is NULL or incorrect, attempting to fix...\n")
			fmt.Printf("[INIT] Current branch_id value: %v, expected: %d\n", dbUser.BranchID, mainBranch.ID)

			// Method 1: Direct SQL update (most reliable)
			if err := s.db.Exec("UPDATE user_models SET branch_id = ? WHERE id = ?", mainBranch.ID, dbUser.ID).Error; err != nil {
				fmt.Printf("[INIT] ERROR: Direct SQL update failed: %v\n", err)
				// Try Method 2: GORM Update
				if err := s.db.Model(&dbUser).Update("branch_id", mainBranch.ID).Error; err != nil {
					fmt.Printf("[INIT] ERROR: GORM Update failed: %v\n", err)
					// Try Method 3: GORM Save
					dbUser.BranchID = &mainBranch.ID
					if err := s.db.Save(&dbUser).Error; err != nil {
						return fmt.Errorf("all branch_id fix methods failed, last error: %w", err)
					}
				}
			}

			// Re-fetch again to verify
			if err := s.db.First(&dbUser, owner.ID).Error; err != nil {
				return fmt.Errorf("failed to re-fetch after fix: %w", err)
			}
			fmt.Printf("[INIT] After fix attempt - BranchID: %v\n", dbUser.BranchID)

			if dbUser.BranchID == nil || *dbUser.BranchID != mainBranch.ID {
				return fmt.Errorf("branch_id fix failed (expected: %d, got: %v)",
					mainBranch.ID, dbUser.BranchID)
			}
			fmt.Printf("[INIT] branch_id fixed successfully: %d\n", *dbUser.BranchID)
		}

		owner = &dbUser // Use the database-fetched user
		// FINAL FIX: Update branch_id directly in database after branch is created
		// This ensures branch_id is set even if it was NULL during user creation
		fmt.Printf("[INIT] Final fix: Updating branch_id for user %v with branch %v\n", owner.ID, mainBranch.ID)
		if err := s.db.Exec("UPDATE user_models SET branch_id = ? WHERE id = ?", mainBranch.ID, owner.ID).Error; err != nil {
			fmt.Printf("[INIT] ERROR: Final branch_id update failed: %v\n", err)
		} else {
			// Verify the update worked
			if err := s.db.First(&dbUser, owner.ID).Error; err == nil {
				fmt.Printf("[INIT] Final verification - BranchID after update: %v\n", dbUser.BranchID)
			}
		}

		fmt.Printf("Owner created successfully: %s (%s) for company: %s, branch: %s (companyId: %v, branchId: %v)\n",
			owner.Username, *owner.Email, company.Name, mainBranch.Name, company.ID, mainBranch.ID)

		// Update branch to link it to the admin user
		updateReq := UpdateBranchRequest{
			AdminUserID: &owner.ID,
		}
		_, err = branchService.UpdateBranch(mainBranch.ID, updateReq)
		if err != nil {
			return fmt.Errorf("failed to update branch with admin user ID: %w", err)
		}
		fmt.Printf("Branch %s linked to admin user: %s (userId: %v)\n", mainBranch.Name, owner.Username, owner.ID)
	} else {
		return fmt.Errorf("failed to check for existing owner: %w", err)
	}

	// Final safety check: Update any super_admin users with NULL branch_id
	// This catches any users that were created before the branch existed
	fmt.Printf("[INIT] ===== FINAL SAFETY CHECK =====\n")
	fmt.Printf("[INIT] Looking for super_admin users with NULL branch_id\n")
	fmt.Printf("[INIT] Branch ID to use: %v\n", mainBranch.ID)

	var adminsWithNullBranch []UserModel
	query := "role = ? AND (branch_id IS NULL OR branch_id = '')"
	if err := s.db.Where(query, SuperAdmin).Find(&adminsWithNullBranch).Error; err == nil {
		fmt.Printf("[INIT] Found %d admin users with NULL branch_id\n", len(adminsWithNullBranch))
		for i := range adminsWithNullBranch {
			if adminsWithNullBranch[i].BranchID == nil {
				fmt.Printf("[INIT] Fixing branch_id for user: %s (id: %v)\n", adminsWithNullBranch[i].Username, adminsWithNullBranch[i].ID)
				// Use direct SQL update to ensure it works
				updateSQL := "UPDATE user_models SET branch_id = ? WHERE id = ?"
				if err := s.db.Exec(updateSQL, mainBranch.ID, adminsWithNullBranch[i].ID).Error; err != nil {
					fmt.Printf("[INIT] ERROR: Failed to update branch_id for admin user %s: %v\n", adminsWithNullBranch[i].Username, err)
				} else {
					fmt.Printf("[INIT] ✓ Successfully fixed branch_id for admin user: %s (branchId: %v)\n", adminsWithNullBranch[i].Username, mainBranch.ID)
					// Verify the update
					var verifyUser UserModel
					if err := s.db.First(&verifyUser, adminsWithNullBranch[i].ID).Error; err == nil {
						fmt.Printf("[INIT] Verification - User %s now has branch_id: %v\n", verifyUser.Username, verifyUser.BranchID)
					}
				}
			}
		}
	} else {
		fmt.Printf("[INIT] Error finding admins with NULL branch_id: %v\n", err)
	}

	fmt.Printf("[INIT] ===== INITIALIZATION COMPLETE =====\n")
	return nil
}

// Branch Service (moved from Branch package to avoid import cycle)
var branchService *BranchService

type BranchService struct {
	db *gorm.DB
}

func NewBranchService() *BranchService {
	return &BranchService{}
}

// InitializeBranchService initializes the branch service with a database connection
func InitializeBranchService(db *gorm.DB) {
	branchService = &BranchService{db: db}
}

// GetBranchService returns the initialized branch service
func GetBranchService() *BranchService {
	return branchService
}

func (s *BranchService) GetBranchByID(id uint) (*Branch, error) {
	var branch Branch
	if err := s.db.First(&branch, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}
	return &branch, nil
}

func (s *BranchService) GetBranchesByCompany(companyID uint) ([]*Branch, error) {
	var branches []*Branch
	if err := s.db.Where("company_id = ?", companyID).Find(&branches).Error; err != nil {
		return nil, err
	}
	return branches, nil
}

func (s *BranchService) GetAllBranches() ([]*Branch, error) {
	var branches []*Branch
	if err := s.db.Find(&branches).Error; err != nil {
		return nil, err
	}
	return branches, nil
}

func (s *BranchService) CreateBranch(req CreateBranchRequest) (*Branch, error) {
	branch := &Branch{
		CompanyID:   req.CompanyID,
		AdminUserID: req.AdminUserID,
		Name:        req.Name,
		Address:     req.Address,
		Phone:       req.Phone,
	}

	if err := s.db.Create(branch).Error; err != nil {
		return nil, err
	}

	return branch, nil
}

func (s *BranchService) UpdateBranch(id uint, req UpdateBranchRequest) (*Branch, error) {
	var branch Branch
	if err := s.db.First(&branch, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("branch not found")
		}
		return nil, err
	}

	if req.CompanyID != nil {
		branch.CompanyID = *req.CompanyID
	}
	if req.AdminUserID != nil {
		branch.AdminUserID = req.AdminUserID
	}
	if req.Name != nil {
		branch.Name = *req.Name
	}
	if req.Address != nil {
		branch.Address = req.Address
	}
	if req.Phone != nil {
		branch.Phone = req.Phone
	}

	if err := s.db.Save(&branch).Error; err != nil {
		return nil, err
	}

	return &branch, nil
}

func (s *BranchService) DeleteBranch(id uint) error {
	var branch Branch
	if err := s.db.First(&branch, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("branch not found")
		}
		return err
	}

	if err := s.db.Delete(&branch).Error; err != nil {
		return err
	}

	return nil
}

// InitializeMockData creates initial mock users (for development/testing)
func (s *UserService) InitializeMockData() {
	// Only initialize if database is empty
	var count int64
	s.db.Model(&UserModel{}).Count(&count)
	if count > 0 {
		return
	}

	// Mock users initialization removed - users should be created through proper Company/Branch structure
	// If you need test data, create it through the API with proper Company and Branch IDs
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	// Use bcrypt.DefaultCost (10) - good balance between security and performance
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// populateBranchAndCompany populates the branch and company fields from BranchID
// Company is always derived from the branch
func (s *UserService) populateBranchAndCompany(user *UserModel) {
	if user.BranchID != nil {
		var branch Branch
		if err := s.db.First(&branch, "id = ?", *user.BranchID).Error; err == nil {
			user.Branch = branch.Name
			// Get company ID and name from branch's company
			user.CompanyID = &branch.CompanyID
			var company Company.Company
			if err := s.db.First(&company, "id = ?", branch.CompanyID).Error; err == nil {
				user.Company = company.Name
			}
		}
	}
}

// GetCompanyIDFromBranch returns the company ID for a given branch ID
func (s *UserService) GetCompanyIDFromBranch(branchID uint) (*uint, error) {
	var branch Branch
	if err := s.db.First(&branch, "id = ?", branchID).Error; err != nil {
		return nil, err
	}
	return &branch.CompanyID, nil
}

// GetNotificationPreferences retrieves notification preferences for a user
// Creates default preferences if they don't exist
func (s *UserService) GetNotificationPreferences(userID uint) (*NotificationPreferences, error) {
	var prefs NotificationPreferences
	err := s.db.Where("user_id = ?", userID).First(&prefs).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create default preferences if they don't exist
		prefs = NotificationPreferences{
			UserID:              userID,
			SalesNotifications:  true,
			InventoryAlerts:     true,
			UserActivity:        false,
			SystemUpdates:       true,
			EmailNotifications:  false,
			PushNotifications:   true,
		}
		if err := s.db.Create(&prefs).Error; err != nil {
			return nil, err
		}
		return &prefs, nil
	}
	
	if err != nil {
		return nil, err
	}
	
	return &prefs, nil
}

// UpdateNotificationPreferences updates notification preferences for a user
// Creates default preferences if they don't exist
func (s *UserService) UpdateNotificationPreferences(userID uint, req UpdateNotificationPreferencesRequest) (*NotificationPreferences, error) {
	var prefs NotificationPreferences
	err := s.db.Where("user_id = ?", userID).First(&prefs).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new preferences with defaults
		prefs = NotificationPreferences{
			UserID:              userID,
			SalesNotifications:  true,
			InventoryAlerts:     true,
			UserActivity:        false,
			SystemUpdates:       true,
			EmailNotifications:  false,
			PushNotifications:   true,
		}
	}
	
	// Update only the fields that are provided
	if req.SalesNotifications != nil {
		prefs.SalesNotifications = *req.SalesNotifications
	}
	if req.InventoryAlerts != nil {
		prefs.InventoryAlerts = *req.InventoryAlerts
	}
	if req.UserActivity != nil {
		prefs.UserActivity = *req.UserActivity
	}
	if req.SystemUpdates != nil {
		prefs.SystemUpdates = *req.SystemUpdates
	}
	if req.EmailNotifications != nil {
		prefs.EmailNotifications = *req.EmailNotifications
	}
	if req.PushNotifications != nil {
		prefs.PushNotifications = *req.PushNotifications
	}
	
	if err := s.db.Save(&prefs).Error; err != nil {
		return nil, err
	}
	
	return &prefs, nil
}
