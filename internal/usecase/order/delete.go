package order

import (
	"context"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

func (s *service) Delete(ctx context.Context, userID string, id string) error {
	if userID == "" {
		return entity.ErrUnauthorized
	}
	if id == "" {
		return entity.ErrInvalidID
	}
	// Ownership is checked inside repository when marking deleted for safety,
	// but we keep consistent flow: fetch, check, then mark
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if o == nil || o.IsDeleted {
		return entity.ErrNotFound
	}
	if o.UserID != userID {
		return entity.ErrForeignOwnership
	}
	if err := s.repo.MarkDeleted(ctx, id, userID); err != nil {
		return err
	}
	_ = s.producer.OrderDeleted(ctx, id, userID)
	return nil
}
