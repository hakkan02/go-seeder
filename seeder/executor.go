package seeder

import (
	"context"
	"database/sql"
	"fmt"
)

// Executor is responsible for executing seed files against the database.
type Executor struct{}

// NewExecutor creates and returns a new Executor instance.
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute runs the given seed in the specified direction.
func (e *Executor) Execute(ctx context.Context, tx *sql.Tx, seed Seed, direction Direction) error {
	var query string

	switch direction {
	case Up:
		query = seed.Up
	case Down:
		query = seed.Down
	default:
		return fmt.Errorf(
			"unsupported seed direction: %s",
			direction,
		)
	}

	if query == "" {
		return fmt.Errorf(
			"seed %d has no SQL for direction %s",
			seed.Version,
			direction,
		)
	}

	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf(
			"execute seed %d (%s): %w",
			seed.Version,
			direction,
			err,
		)
	}
	return nil
}
