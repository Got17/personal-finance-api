package transfer

import (
	"context"
	"errors"

	"github.com/BounkhongDev/bkgo/contract"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
		var dummy string
		if tx.Dialector != nil && tx.Dialector.Name() == "postgres" {
			if err := tx.Table("accounts").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", t.SourceAccountID).Select("id").Take(&dummy).Error; err != nil {
				return err
			}
		}

		sourceBalance, err := r.calculateAccountBalance(tx, t.UserID, t.SourceAccountID)
		if err != nil {
			return err
		}

		requiredSourceOutlay := t.SourceAmountMinor
		if fee != nil && fee.AccountID == t.SourceAccountID {
			requiredSourceOutlay += fee.AmountMinor
		}

		if sourceBalance < requiredSourceOutlay {
			return ErrInsufficientAccountBalance
		}

		if fee != nil && fee.AccountID != t.SourceAccountID {
			if tx.Dialector != nil && tx.Dialector.Name() == "postgres" {
				if err := tx.Table("accounts").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", fee.AccountID).Select("id").Take(&dummy).Error; err != nil {
					return err
				}
			}
			feeBalance, err := r.calculateAccountBalance(tx, fee.UserID, fee.AccountID)
			if err != nil {
				return err
			}
			if feeBalance < fee.AmountMinor {
				return ErrInsufficientFeeAccountBalance
			}
		}

		if quote != nil {
			if err := tx.Create(quote).Error; err != nil {
				return err
			}
		}
		if err := tx.Omit("HistoricalFXQuote").Create(t).Error; err != nil {
			return err
		}
		if fee != nil {
			if err := tx.Create(fee).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *transferRepository) GetAccountBalance(ctx context.Context, userID string, accountID string) (int64, error) {
	return r.calculateAccountBalance(r.db.Session(ctx), userID, accountID)
}

func (r *transferRepository) calculateAccountBalance(tx *gorm.DB, userID, accountID string) (int64, error) {
	var balance int64
	query := `
		SELECT 
			COALESCE(SUM(CASE 
				WHEN account_id = ? AND kind = 'income' THEN amount_minor 
				WHEN destination_account_id = ? AND kind = 'transfer' THEN destination_amount_minor 
				ELSE 0 
			END), 0)
			-
			COALESCE(SUM(CASE 
				WHEN account_id = ? AND kind = 'expense' THEN amount_minor 
				WHEN account_id = ? AND kind = 'transfer' THEN amount_minor 
				ELSE 0 
			END), 0) AS balance
		FROM financial_records
		WHERE user_id = ? AND is_active = true AND (account_id = ? OR destination_account_id = ?)
	`
	if err := tx.Raw(query, accountID, accountID, accountID, accountID, userID, accountID, accountID).Scan(&balance).Error; err != nil {
		return 0, err
	}
	return balance, nil
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

	var feeIDs []string
	for _, item := range list {
		if item.TransferFeeRecordID != nil {
			feeIDs = append(feeIDs, *item.TransferFeeRecordID)
		}
	}
	if len(feeIDs) > 0 {
		var feeRecords []financialrecord.FinancialRecord
		if err := r.db.Session(ctx).Where("id IN ?", feeIDs).Find(&feeRecords).Error; err == nil {
			feeMap := make(map[string]*TransferFeeResult, len(feeRecords))
			for i := range feeRecords {
				feeMap[feeRecords[i].ID] = toFeeResult(&feeRecords[i])
			}
			for _, item := range list {
				if item.TransferFeeRecordID != nil {
					item.TransferFee = feeMap[*item.TransferFeeRecordID]
				}
			}
		}
	}

	return list, nil
}
