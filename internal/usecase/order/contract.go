package order

//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

import (
	"context"
	"time"

	"github.com/nikolaev/service-order/internal/domain/entity"
)

type Repository interface {
	Create(ctx context.Context, o *entity.Order) error
	GetByID(ctx context.Context, id string) (*entity.Order, error)
	Update(ctx context.Context, o *entity.Order) error
	MarkDeleted(ctx context.Context, id string, userID string) error
	ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error)
}

type Producer interface {
	OrderCreated(ctx context.Context, o *entity.Order) error
	OrderUpdated(ctx context.Context, o *entity.Order) error
	OrderDeleted(ctx context.Context, id string, userID string) error
}

type Service interface {
	Create(ctx context.Context, userID string, in CreateInput) (*entity.Order, error)
	Get(ctx context.Context, userID string, id string) (*entity.Order, error)
	GetStatus(ctx context.Context, userID string, id string) (entity.OrderStatus, error)
	ListFrom(ctx context.Context, from time.Time) ([]*entity.Order, error)
	Update(ctx context.Context, userID string, id string, in UpdateInput) (*entity.Order, error)
	Delete(ctx context.Context, userID string, id string) error
}

type CreateInput struct {
	OrderNumber  string
	FIO          string
	RestaurantID string
	Items        []entity.Item
	TotalPrice   int64
	Address      entity.DeliveryAddress
}

type UpdateInput struct {
	OrderNumber *string
	FIO         *string
	Items       *[]entity.Item
	TotalPrice  *int64
	Address     *entity.DeliveryAddress
}

// Clock abstracts time source for testability and guideline compliance.
type Clock interface{ Now() time.Time }

// log is a minimal structured log interface per guidelines.
type log interface {
	WithFields(ctx context.Context, fields map[string]any) context.Context
	Info(ctx context.Context, args ...any)
}

// metric is a minimal metric interface per guidelines.
type metric interface{ Increment(key string) }

type service struct {
	repo     Repository
	producer Producer
	clock    Clock
	log      log
	metric   metric
}

// New provides default dependencies (system clock and no-op log/metric) for convenience.
func New(repo Repository, producer Producer) Service {
	return NewWithDeps(repo, producer, systemClock{}, noopLog{}, noopMetric{})
}

// NewWithDeps allows injecting clock/log/metric; prefer in tests and composition roots.
func NewWithDeps(repo Repository, producer Producer, clk Clock, l log, m metric) Service {
	return &service{repo: repo, producer: producer, clock: clk, log: l, metric: m}
}

// advanceStatus calculates next statuses over time windows without scheduler, based on CreatedAt
func advanceStatus(now time.Time, o *entity.Order) {
	dur := now.Sub(o.CreatedAt)
	switch {
	case o.Status == entity.OrderStatusCanceled || o.Status == entity.OrderStatusDeleted:
		return
	case dur >= 10*time.Minute:
		o.Status = entity.OrderStatusDelivering
	case dur >= 1*time.Minute:
		if o.Status == entity.OrderStatusPending || o.Status == entity.OrderStatusCreated {
			o.Status = entity.OrderStatusCooking
		}
	default:
		if o.Status == "" {
			o.Status = entity.OrderStatusPending
		}
	}
}

// systemClock is the default Clock implementation.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// noopLog is a default no-op logger implementation.
type noopLog struct{}

func (noopLog) WithFields(ctx context.Context, fields map[string]any) context.Context { return ctx }
func (noopLog) Info(ctx context.Context, args ...any)                                 {}

// noopMetric is a default no-op metric implementation.
type noopMetric struct{}

func (noopMetric) Increment(key string) {}
