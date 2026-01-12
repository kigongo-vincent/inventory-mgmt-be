package Product

import (
	"errors"

	"gorm.io/gorm"
)

// productsDB removed - using database now

var productService *ProductService

type ProductService struct {
	db *gorm.DB
}

func NewProductService() *ProductService {
	return &ProductService{}
}

// InitializeService initializes the product service with a database connection
func InitializeService(db *gorm.DB) {
	productService = &ProductService{db: db}
	// TODO: Update all methods to use database instead of in-memory storage
}

func (s *ProductService) GetProductByID(id uint) (*Product, error) {
	var product Product
	if err := s.db.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &product, nil
}

func (s *ProductService) GetAllProducts() ([]*Product, error) {
	var products []*Product
	if err := s.db.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (s *ProductService) GetAllProductsByCompany(companyID uint) ([]*Product, error) {
	var products []*Product
	if err := s.db.Where("company_id = ?", companyID).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (s *ProductService) GetProductsByCompany(companyID uint) ([]*Product, error) {
	var products []*Product
	if err := s.db.Where("company_id = ?", companyID).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (s *ProductService) CreateProduct(req CreateProductRequest) (*Product, error) {
	product := &Product{
		Name:       req.Name,
		Price:      req.Price,
		Currency:   req.Currency,
		CompanyID:  req.CompanyID,
		Quantity:   req.Quantity,
		ImageURI:   req.ImageURI,
		Attributes: JSONB(req.Attributes),
		SyncStatus: nil,
	}

	if product.Attributes == nil {
		product.Attributes = make(JSONB)
	}

	if err := s.db.Create(product).Error; err != nil {
		return nil, err
	}

	return product, nil
}

func (s *ProductService) UpdateProduct(id uint, req UpdateProductRequest) (*Product, error) {
	var product Product
	if err := s.db.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Currency != nil {
		product.Currency = *req.Currency
	}
	if req.CompanyID != nil {
		product.CompanyID = *req.CompanyID
	}
	if req.Quantity != nil {
		product.Quantity = *req.Quantity
	}
	if req.ImageURI != nil {
		product.ImageURI = req.ImageURI
	}
	if req.Attributes != nil {
		product.Attributes = JSONB(req.Attributes)
	}

	if err := s.db.Save(&product).Error; err != nil {
		return nil, err
	}

	return &product, nil
}

func (s *ProductService) DeleteProduct(id uint) error {
	var product Product
	if err := s.db.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("product not found")
		}
		return err
	}
	if err := s.db.Delete(&product).Error; err != nil {
		return err
	}
	return nil
}

func (s *ProductService) ReduceProductQuantity(id uint, quantity int) error {
	var product Product
	if err := s.db.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("product not found")
		}
		return err
	}

	if product.Quantity < quantity {
		return errors.New("insufficient quantity")
	}

	product.Quantity -= quantity
	if err := s.db.Save(&product).Error; err != nil {
		return err
	}

	return nil
}
