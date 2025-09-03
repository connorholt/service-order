package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/dig"

	"github.com/nikolaev/service-order/internal/gateway/kafka"
	"github.com/nikolaev/service-order/internal/handlers"
	repo "github.com/nikolaev/service-order/internal/repository/order"
	seed "github.com/nikolaev/service-order/internal/usecase/debug/seed"
	ucase "github.com/nikolaev/service-order/internal/usecase/order"
)

// Switch to in-memory repository, database not used anymore.
// We provide both the concrete repo for workers and the interface for usecases.
func provideInMemory() *repo.InMemory { return repo.NewInMemory() }
func provideRepo(mem *repo.InMemory) ucase.Repository { return mem }

// Provide Kafka producer: if KAFKA_BROKERS is set, use Sarama; otherwise Noop.
func provideProducer() ucase.Producer {
	if os.Getenv("KAFKA_BROKERS") != "" {
		p, err := kafka.NewSaramaProducer()
		if err == nil {
			return p
		}
		log.Printf("failed to init sarama producer, fallback to noop: %v", err)
	}
	return kafka.NoopProducer{}
}

func provideService(r ucase.Repository, p ucase.Producer) ucase.Service {
	return ucase.New(r, p)
}

type sysClock struct{}

func (sysClock) Now() time.Time { return time.Now().UTC() }

func provideSeeder(mem *repo.InMemory) seed.Service {
	return seed.New(mem, sysClock{})
}

// provideOrderHandler ensures the debug seeder is injected into the handler.
func provideOrderHandler(svc ucase.Service, dbg seed.Service) *handlers.OrderHandler {
	return handlers.NewOrderHandler(svc, dbg)
}

func provideRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	return r
}

// runStatusWorker periodically advances orders through statuses according to timings.
func runStatusWorker(ctx context.Context, mem *repo.InMemory, prod ucase.Producer) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			changed := mem.AdvanceStatuses(now)
			for _, o := range changed {
				_ = prod.OrderUpdated(ctx, o)
			}
		}
	}
}

func main() {
	c := dig.New()

	// Register providers
	_ = c.Provide(provideInMemory)
	_ = c.Provide(provideRepo)
	_ = c.Provide(provideProducer)
	_ = c.Provide(provideService)
	_ = c.Provide(provideSeeder)
	_ = c.Provide(provideOrderHandler)
	_ = c.Provide(provideRouter)

	// Build and run HTTP server
 err := c.Invoke(func(r *chi.Mux, h *handlers.OrderHandler, mem *repo.InMemory, prod ucase.Producer) error {
		// background goroutine for status progression
		ctx, cancel := context.WithCancel(context.Background())
		go func() { defer cancel(); runStatusWorker(ctx, mem, prod) }()
		r.Mount("/public/api/v1", h.Routes())
		log.Println("service started on :8080")
		return http.ListenAndServe(":8080", r)
	})
	if err != nil {
		log.Fatal(err)
	}
}
