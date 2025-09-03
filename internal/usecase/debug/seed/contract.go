package seed

//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

import (
	"context"
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

// Repository is a minimal dependency to persist orders.
type Repository interface {
	Create(ctx context.Context, o *entity.Order) error
}

// Clock provides current time (UTC) for determinism and testability.
type Clock interface{ Now() time.Time }

// Service seeds debug data.
type Service interface {
	// Seed creates n orders for the user and returns them.
	Seed(ctx context.Context, userID string, n int) ([]*entity.Order, error)
}

type service struct {
	repo  Repository
	clk   Clock
}

func New(repo Repository, clk Clock) Service { return &service{repo: repo, clk: clk} }
