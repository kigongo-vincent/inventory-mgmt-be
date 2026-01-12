package Company

import (
	"errors"

	"gorm.io/gorm"
)

var companyService *CompanyService

type CompanyService struct {
	db *gorm.DB
}

func NewCompanyService() *CompanyService {
	return &CompanyService{}
}

// InitializeService initializes the company service with a database connection
func InitializeService(db *gorm.DB) {
	companyService = &CompanyService{db: db}
}

// GetCompanyService returns the initialized company service
func GetCompanyService() *CompanyService {
	return companyService
}

func (s *CompanyService) GetCompanyByID(id uint) (*Company, error) {
	var company Company
	if err := s.db.First(&company, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("company not found")
		}
		return nil, err
	}
	return &company, nil
}

func (s *CompanyService) GetCompanyByEmail(email string) (*Company, error) {
	var company Company
	if err := s.db.Where("email = ?", email).First(&company).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("company not found")
		}
		return nil, err
	}
	return &company, nil
}

func (s *CompanyService) GetAllCompanies() ([]*Company, error) {
	var companies []*Company
	if err := s.db.Find(&companies).Error; err != nil {
		return nil, err
	}
	return companies, nil
}

func (s *CompanyService) CreateCompany(req CreateCompanyRequest) (*Company, error) {
	// Check if email already exists
	var existingCompany Company
	if err := s.db.Where("email = ?", req.Email).First(&existingCompany).Error; err == nil {
		return nil, errors.New("company with this email already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	company := &Company{
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Address: req.Address,
	}

	if err := s.db.Create(company).Error; err != nil {
		return nil, err
	}

	return company, nil
}

func (s *CompanyService) UpdateCompany(id uint, req UpdateCompanyRequest) (*Company, error) {
	var company Company
	if err := s.db.First(&company, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("company not found")
		}
		return nil, err
	}

	if req.Name != nil {
		company.Name = *req.Name
	}
	if req.Email != nil {
		// Check email uniqueness
		var existingCompany Company
		if err := s.db.Where("email = ? AND id != ?", *req.Email, id).First(&existingCompany).Error; err == nil {
			return nil, errors.New("company with this email already exists")
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		company.Email = *req.Email
	}
	if req.Phone != nil {
		company.Phone = req.Phone
	}
	if req.Address != nil {
		company.Address = req.Address
	}

	if err := s.db.Save(&company).Error; err != nil {
		return nil, err
	}

	return &company, nil
}

func (s *CompanyService) DeleteCompany(id uint) error {
	var company Company
	if err := s.db.First(&company, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("company not found")
		}
		return err
	}

	if err := s.db.Delete(&company).Error; err != nil {
		return err
	}

	return nil
}
