package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
)

// Config holds the configuration for the Seeder.
// DB is the database connection.
// FS is the filesystem containing the seed files.
type Config struct {
	DB *sql.DB
	FS fs.FS
}

// Seeder is the main entry point for managing database seeds.
type Seeder struct {
	orchestrator *Orchestrator
}

// New creates and returns a new Seeder instance. 
// It initializes the seed history and discovers seeds from the provided filesystem.
func New(ctx context.Context, cfg Config) (*Seeder, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if cfg.FS == nil {
		return nil, fmt.Errorf("seed filesystem is required")
	}

	seeds, err := DiscoverFS(cfg.FS)
	if err != nil {
		return nil, fmt.Errorf("discover seeds: %w", err)
	}

	history := NewHistory(cfg.DB)

	if err := history.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize history: %w", err)
	}
	return &Seeder{
		orchestrator: NewOrchestrator(seeds, history, NewExecutor()),
	}, nil
}

// Up executes pending seeds.
func (s *Seeder) Up(ctx context.Context, steps int) error {
	return s.orchestrator.Up(ctx, steps)
}

// Down rolls back the specified number of seeds.
func (s *Seeder) Down(ctx context.Context, steps int) error {
	return s.orchestrator.Down(ctx, steps)
}

// Reset rolls back all applied seeds.
func (s *Seeder) Reset(ctx context.Context) error {
	return s.orchestrator.Reset(ctx)
}

// Version returns the current seed state.
func (s *Seeder) Version(ctx context.Context) (State, error) {
	return s.orchestrator.Version(ctx)
}