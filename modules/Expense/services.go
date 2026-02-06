package Expense

import (
	"errors"
	"time"

	"gorm.io/gorm"
	Branch "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Branch"
	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
)

var expenseService *ExpenseService

type ExpenseService struct {
	db *gorm.DB
}

func NewExpenseService() *ExpenseService {
	return &ExpenseService{}
}

func InitializeService(db *gorm.DB) {
	expenseService = &ExpenseService{db: db}
}

func GetExpenseService() *ExpenseService {
	return expenseService
}

func (s *ExpenseService) GetExpenseByID(id uint) (*Expense, error) {
	var expense Expense
	if err := s.db.First(&expense, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("expense not found")
		}
		return nil, err
	}
	s.populateRelations(&expense)
	return &expense, nil
}

func (s *ExpenseService) GetAllExpenses() ([]*Expense, error) {
	var expenses []*Expense
	if err := s.db.Find(&expenses).Error; err != nil {
		return nil, err
	}
	for i := range expenses {
		s.populateRelations(expenses[i])
	}
	return expenses, nil
}

func (s *ExpenseService) GetExpensesByUser(userID uint) ([]*Expense, error) {
	var expenses []*Expense
	if err := s.db.Where("user_id = ?", userID).Find(&expenses).Error; err != nil {
		return nil, err
	}
	for i := range expenses {
		s.populateRelations(expenses[i])
	}
	return expenses, nil
}

func (s *ExpenseService) GetExpensesByBranch(branchID uint) ([]*Expense, error) {
	var expenses []*Expense
	if err := s.db.Where("branch_id = ?", branchID).Find(&expenses).Error; err != nil {
		return nil, err
	}
	for i := range expenses {
		s.populateRelations(expenses[i])
	}
	return expenses, nil
}

func (s *ExpenseService) GetExpensesByDateRange(startDate, endDate time.Time) ([]*Expense, error) {
	var expenses []*Expense
	if err := s.db.Where("created_at >= ? AND created_at <= ?", startDate, endDate).Find(&expenses).Error; err != nil {
		return nil, err
	}
	for i := range expenses {
		s.populateRelations(expenses[i])
	}
	return expenses, nil
}

func (s *ExpenseService) CreateExpense(req CreateExpenseRequest) (*Expense, error) {
	expense := &Expense{
		Amount:      req.Amount,
		Description: req.Description,
		Category:    req.Category,
		Currency:    req.Currency,
		UserID:      req.UserID,
		BranchID:    req.BranchID,
	}

	if err := s.db.Create(expense).Error; err != nil {
		return nil, err
	}

	s.populateRelations(expense)
	return expense, nil
}

func (s *ExpenseService) UpdateExpense(id uint, req UpdateExpenseRequest) (*Expense, error) {
	var expense Expense
	if err := s.db.First(&expense, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("expense not found")
		}
		return nil, err
	}

	if req.Amount != nil {
		expense.Amount = *req.Amount
	}
	if req.Description != nil {
		expense.Description = *req.Description
	}
	if req.Category != nil {
		expense.Category = *req.Category
	}
	if req.Currency != nil {
		expense.Currency = *req.Currency
	}

	if err := s.db.Save(&expense).Error; err != nil {
		return nil, err
	}

	s.populateRelations(&expense)
	return &expense, nil
}

func (s *ExpenseService) DeleteExpense(id uint) error {
	var expense Expense
	if err := s.db.First(&expense, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("expense not found")
		}
		return err
	}
	return s.db.Delete(&expense).Error
}

func (s *ExpenseService) populateRelations(expense *Expense) {
	if expense.UserID != 0 {
		user, err := User.GetUserService().GetUserByID(expense.UserID)
		if err == nil && user != nil {
			expense.User = &UserResponse{
				ID:   user.ID,
				Name: user.Name,
			}
		}
	}
	if expense.BranchID != 0 {
		var branch Branch.Branch
		if err := s.db.First(&branch, "id = ?", expense.BranchID).Error; err == nil {
			expense.Branch = &BranchResponse{
				ID:   branch.ID,
				Name: branch.Name,
			}
		}
	}
}
