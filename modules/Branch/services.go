package Branch

import (
	"errors"

	"gorm.io/gorm"
)

var branchService *BranchService

type BranchService struct {
	db *gorm.DB
}

func NewBranchService() *BranchService {
	return &BranchService{}
}

// InitializeService initializes the branch service with a database connection
func InitializeService(db *gorm.DB) {
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
