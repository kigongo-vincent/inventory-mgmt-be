package Sale

import (
	"errors"
	"time"

	"gorm.io/gorm"
	Branch "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Branch"
	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
)

// salesDB removed - using database now

var saleService *SaleService

type SaleService struct {
	db *gorm.DB
}

// GetDB returns the database connection (for use in router)
func (s *SaleService) GetDB() *gorm.DB {
	return s.db
}

func NewSaleService() *SaleService {
	return &SaleService{}
}

// InitializeService initializes the sale service with a database connection
func InitializeService(db *gorm.DB) {
	saleService = &SaleService{db: db}
	// TODO: Update all methods to use database instead of in-memory storage
}

func (s *SaleService) GetSaleByID(id uint) (*Sale, error) {
	var sale Sale
	if err := s.db.First(&sale, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("sale not found")
		}
		return nil, err
	}
	s.populateSeller(&sale)
	return &sale, nil
}

func (s *SaleService) GetAllSales() ([]*Sale, error) {
	var sales []*Sale
	if err := s.db.Find(&sales).Error; err != nil {
		return nil, err
	}
	for i := range sales {
		s.populateSeller(sales[i])
	}
	return sales, nil
}

func (s *SaleService) GetSalesByUser(userID uint) ([]*Sale, error) {
	var sales []*Sale
	if err := s.db.Where("seller_id = ?", userID).Find(&sales).Error; err != nil {
		return nil, err
	}
	for i := range sales {
		s.populateSeller(sales[i])
	}
	return sales, nil
}

func (s *SaleService) GetSalesByBranch(branchID uint) ([]*Sale, error) {
	// Get sales by finding users in the branch, then their sales
	var sales []*Sale
	if err := s.db.Joins("JOIN user_models ON sales.seller_id = user_models.id").
		Where("user_models.branch_id = ?", branchID).
		Find(&sales).Error; err != nil {
		return nil, err
	}
	for i := range sales {
		s.populateSeller(sales[i])
	}
	return sales, nil
}

func (s *SaleService) GetSalesByDateRange(startDate, endDate time.Time) ([]*Sale, error) {
	var sales []*Sale
	if err := s.db.Where("created_at >= ? AND created_at <= ?", startDate, endDate).Find(&sales).Error; err != nil {
		return nil, err
	}
	for i := range sales {
		s.populateSeller(sales[i])
	}
	return sales, nil
}

func (s *SaleService) CreateSale(req CreateSaleRequest) (*Sale, error) {
	// Use provided total price (allows manual override from frontend)
	// Frontend sends calculated total or overridden total
	finalTotalPrice := req.TotalPrice
	
	sale := &Sale{
		ProductID:         req.ProductID,
		ProductName:       req.ProductName,
		ProductAttributes: JSONB(req.ProductAttributes),
		Quantity:          req.Quantity,
		UnitPrice:         req.UnitPrice,
		ExtraCosts:        req.ExtraCosts,
		TotalPrice:        finalTotalPrice, // Use provided total (calculated or overridden)
		Currency:          req.Currency,
		SellerID:          req.SellerID,
		PaymentStatus:     req.PaymentStatus,
		BuyerName:         req.BuyerName,
		BuyerContact:      req.BuyerContact,
		BuyerLocation:     req.BuyerLocation,
		SyncStatus:        nil,
	}

	if sale.ProductAttributes == nil {
		sale.ProductAttributes = make(JSONB)
	}

	if err := s.db.Create(sale).Error; err != nil {
		return nil, err
	}

	// Populate seller and branch information before returning
	s.populateSeller(sale)
	return sale, nil
}

func (s *SaleService) UpdateSale(id uint, req UpdateSaleRequest) (*Sale, error) {
	var sale Sale
	if err := s.db.First(&sale, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("sale not found")
		}
		return nil, err
	}

	if req.ProductID != nil {
		sale.ProductID = *req.ProductID
	}
	if req.ProductName != nil {
		sale.ProductName = *req.ProductName
	}
	if req.ProductAttributes != nil {
		sale.ProductAttributes = JSONB(req.ProductAttributes)
	}
	if req.Quantity != nil {
		sale.Quantity = *req.Quantity
	}
	if req.UnitPrice != nil {
		sale.UnitPrice = *req.UnitPrice
	}
	if req.ExtraCosts != nil {
		sale.ExtraCosts = *req.ExtraCosts
	}
	if req.TotalPrice != nil {
		sale.TotalPrice = *req.TotalPrice
	} else if req.UnitPrice != nil || req.Quantity != nil || req.ExtraCosts != nil {
		// Recalculate total if any component changed
		unitPrice := sale.UnitPrice
		quantity := sale.Quantity
		extraCosts := sale.ExtraCosts
		if req.UnitPrice != nil {
			unitPrice = *req.UnitPrice
		}
		if req.Quantity != nil {
			quantity = *req.Quantity
		}
		if req.ExtraCosts != nil {
			extraCosts = *req.ExtraCosts
		}
		sale.TotalPrice = (unitPrice * float64(quantity)) + extraCosts
	}
	if req.Currency != nil {
		sale.Currency = *req.Currency
	}
	if req.SellerID != nil {
		sale.SellerID = *req.SellerID
	}
	if req.PaymentStatus != nil {
		sale.PaymentStatus = *req.PaymentStatus
	}
	if req.BuyerName != nil {
		sale.BuyerName = req.BuyerName
	}
	if req.BuyerContact != nil {
		sale.BuyerContact = req.BuyerContact
	}
	if req.BuyerLocation != nil {
		sale.BuyerLocation = req.BuyerLocation
	}

	if err := s.db.Save(&sale).Error; err != nil {
		return nil, err
	}

	return &sale, nil
}

func (s *SaleService) DeleteSale(id uint) error {
	var sale Sale
	if err := s.db.First(&sale, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("sale not found")
		}
		return err
	}
	if err := s.db.Delete(&sale).Error; err != nil {
		return err
	}
	return nil
}

// populateSeller populates seller information from FK relationship
func (s *SaleService) populateSeller(sale *Sale) {
	if sale.SellerID == 0 {
		return
	}
	
	user, err := User.GetUserService().GetUserByID(sale.SellerID)
	if err == nil && user != nil {
		sale.Seller = &UserResponse{
			ID:   user.ID,
			Name: user.Name,
		}
		// Populate branch from seller's branch
		if user.BranchID != nil {
			var branch Branch.Branch
			if err := s.db.First(&branch, "id = ?", *user.BranchID).Error; err == nil {
				sale.Branch = &BranchResponse{
					ID:   branch.ID,
					Name: branch.Name,
				}
			}
		}
	}
}

// GetCompanyIDFromSale returns the company ID for a given sale
func (s *SaleService) GetCompanyIDFromSale(sale *Sale) (*uint, error) {
	if sale.SellerID == 0 {
		return nil, errors.New("sale has no seller")
	}
	
	user, err := User.GetUserService().GetUserByID(sale.SellerID)
	if err != nil {
		return nil, err
	}
	
	if user.BranchID == nil {
		return nil, errors.New("seller has no branch")
	}
	
	companyID, err := User.GetUserService().GetCompanyIDFromBranch(*user.BranchID)
	if err != nil {
		return nil, err
	}
	
	return companyID, nil
}
