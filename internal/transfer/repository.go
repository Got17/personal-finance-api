package transfer

import (
	"context"
	"errors"

	"github.com/BounkhongDev/bkgo/contract"
	"gorm.io/gorm"

	"github.com/Got17/personal-finance-api/internal/financialrecord"
	"github.com/Got17/personal-finance-api/internal/fxquote"
)

type transferRepository struct {
	db contract.ORM
}

func NewTransferRepository(db contract.ORM) TransferRepository {
	return &transferRepository{db: db}
}

func (r *transferRepository) CreateTransferWithLegsAndFee(ctx context.Context, t *Transfer, quote *fxquote.HistoricalFXQuote, fee *financialrecord.FinancialRecord) error {
	return r.db.Session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		if quote != nil {
			if err := tx.Create(quote).Error; err != nil {
				return err
			}
		}
		if fee != nil {
			if err := tx.Create(fee).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *transferRepository) FindByID(ctx context.Context, id string) (*Transfer, error) {
	var t Transfer
	if err := r.db.Session(ctx).Preload("HistoricalFXQuote").First(&t, "id = ? AND kind = ?", id, "transfer").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransferNotFound
		}
		return nil, err
	}

	if t.TransferFeeRecordID != nil {
		var feeRec financialrecord.FinancialRecord
		if err := r.db.Session(ctx).First(&feeRec, "id = ?", *t.TransferFeeRecordID).Error; err == nil {
			t.TransferFee = toFeeResult(&feeRec)
		}
	}

	return &t, nil
}

func (r *transferRepository) FindByUserID(ctx context.Context, userID string, filter ListTransferFilter) ([]*Transfer, error) {
	query := r.db.Session(ctx).Where("user_id = ? AND kind = ?", userID, "transfer")
	if !filter.IncludeArchived {
		query = query.Where("is_active = ?", true)
	}
	if filter.StartDate != nil {
		query = query.Where("date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("date <= ?", *filter.EndDate)
	}
	if filter.AccountID != "" {
		query = query.Where("account_id = ? OR destination_account_id = ?", filter.AccountID, filter.AccountID)
	}

	var list []*Transfer
	if err := query.Preload("HistoricalFXQuote").Order("date desc").Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}

	for _, item := range list {
		if item.TransferFeeRecordID != nil {
			var feeRec financialrecord.FinancialRecord
			if err := r.db.Session(ctx).First(&feeRec, "id = ?", *item.TransferFeeRecordID).Error; err == nil {
				item.TransferFee = toFeeResult(&feeRec)
			}
		}
	}

	return list, nil
}
