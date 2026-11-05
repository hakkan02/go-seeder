package seeder

import (
	"context"
	"database/sql"
	"fmt"
)

type Executor struct {
	db *sql.DB
}

func NewExecutor(db *sql.DB) *Executor {
	return &Executor{
		db: db,
	}
}

func (e *Executor) Execute(ctx context.Context, seed Seed, direction Direction) error {
	var query string

	switch direction {
	case Up:
		query = seed.Up
	case Down:
		query = seed.Down
	default:
		return fmt.Errorf("unsupported seed direction: %s", direction)
	}

	if query == "" {
		return fmt.Errorf(
			"seed %d has no SQL for direction %s",
			seed.Version,
			direction,
		)
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"failed to begin transaction for seed %d: %w",
			seed.Version,
			err,
		)
	}

	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf(
			"failed to execute seed %d (%s): %w",
			seed.Version,
			direction,
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"failed to commit seed %d: %w",
			seed.Version,
			err,
		)
	}

	return nil
}
