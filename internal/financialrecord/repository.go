package financialrecord

import (
	"context"
	"errors"

	"github.com/BounkhongDev/bkgo/contract"
	"gorm.io/gorm"
)

type financialRecordRepository struct{ db contract.ORM }

func NewFinancialRecordRepository(db contract.ORM) FinancialRecordRepository {
	return &financialRecordRepository{db: db}
}

func (r *financialRecordRepository) Create(ctx context.Context, record *FinancialRecord) error {
	return r.db.Session(ctx).Create(record).Error
}

func (r *financialRecordRepository) FindByID(ctx context.Context, id string) (*FinancialRecord, error) {
	var record FinancialRecord
	if err := r.db.Session(ctx).Preload("HistoricalFXQuote").First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFinancialRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r *financialRecordRepository) FindByUserID(ctx context.Context, userID string, filter ListFilter) ([]*FinancialRecord, error) {
	query := r.db.Session(ctx).Where("user_id = ?", userID)
	if !filter.IncludeArchived {
		query = query.Where("is_active = ?", true)
	}
	if filter.StartDate != nil {
		query = query.Where("date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("date <= ?", *filter.EndDate)
	}
	if filter.Kind != "" {
		query = query.Where("kind = ?", filter.Kind)
	}
	if filter.AccountID != "" {
		query = query.Where("account_id = ? OR destination_account_id = ?", filter.AccountID, filter.AccountID)
	}
	if filter.CategoryID != "" {
		query = query.Where("category_id = ?", filter.CategoryID)
	}
	var records []*FinancialRecord
	if err := query.Preload("HistoricalFXQuote").Order("date desc").Order("created_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *financialRecordRepository) Update(ctx context.Context, record *FinancialRecord) error {
	return r.db.Session(ctx).Save(record).Error
}
