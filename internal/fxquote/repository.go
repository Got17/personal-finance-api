package fxquote

import (
	"context"
	"errors"

	"github.com/BounkhongDev/bkgo/contract"
	"gorm.io/gorm"
)

type fxQuoteRepository struct {
	db contract.ORM
}

func NewFXQuoteRepository(db contract.ORM) FXQuoteRepository {
	return &fxQuoteRepository{db: db}
}

func (r *fxQuoteRepository) Create(ctx context.Context, quote *HistoricalFXQuote) error {
	return r.db.Session(ctx).Create(quote).Error
}

func (r *fxQuoteRepository) FindByID(ctx context.Context, id string) (*HistoricalFXQuote, error) {
	var quote HistoricalFXQuote
	if err := r.db.Session(ctx).First(&quote, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFXQuoteNotFound
		}
		return nil, err
	}
	return &quote, nil
}

func (r *fxQuoteRepository) FindByRecordID(ctx context.Context, recordID string) (*HistoricalFXQuote, error) {
	var quote HistoricalFXQuote
	if err := r.db.Session(ctx).First(&quote, "record_id = ?", recordID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFXQuoteNotFound
		}
		return nil, err
	}
	return &quote, nil
}
