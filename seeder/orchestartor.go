
package seeder

import (
	"context"
	"database/sql"
	"fmt"
)

type Orchestrator struct {
	seeds    []Seed
	history  *History
	executor *Executor
}

func NewOrchestrator(ctx context.Context, db *sql.DB, directory string) (*Orchestrator, error) {
	seeds, err := Discover(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to discover seeds: %w", err)
	}

	history := &History{db: db}

	if err := history.Initialize(ctx); err != nil {
		return nil, err
	}

	return &Orchestrator{
		seeds:    seeds,
		history:  history,
		executor: NewExecutor(db),
	}, nil
}

// Up executes all pending seeds in ascending version order.
func (o *Orchestrator) Up(ctx context.Context) error {
	state, err := o.history.Current(ctx)
	if err != nil {
		return err
	}

	if state.Dirty {
		return fmt.Errorf(
			"database is in dirty state at version %d",
			state.Version,
		)
	}

	for _, seed := range o.seeds {
		if uint64(seed.Version) <= state.Version {
			continue
		}

		if err := o.apply(ctx, seed, Up); err != nil {
			return err
		}
	}

	return nil
}

// Down rolls back the specified number of applied seeds.
func (o *Orchestrator) Down(ctx context.Context, steps int) error {
	if steps <= 0 {
		return fmt.Errorf("rollback steps must be greater than zero")
	}

	for range steps {
		state, err := o.history.Current(ctx)
		if err != nil {
			return err
		}

		if state.Version == 0 {
			return nil
		}

		if state.Dirty {
			return fmt.Errorf(
				"database is in dirty state at version %d",
				state.Version,
			)
		}

		seed, found := o.findSeed(state.Version)
		if !found {
			return fmt.Errorf(
				"seed version %d not found",
				state.Version,
			)
		}

		if err := o.apply(ctx, seed, Down); err != nil {
			return err
		}
	}

	return nil
}

// Reset rolls back all applied seeds.
func (o *Orchestrator) Reset(ctx context.Context) error {
	for {
		state, err := o.history.Current(ctx)
		if err != nil {
			return err
		}

		if state.Version == 0 {
			return nil
		}

		if err := o.Down(ctx, 1); err != nil {
			return err
		}
	}
}

// Version returns the current seed version and dirty state.
func (o *Orchestrator) Version(ctx context.Context) (State, error) {
	return o.history.Current(ctx)
}

// apply coordinates the execution and history updates.
func (o *Orchestrator) apply(ctx context.Context, seed Seed, direction Direction) error {
	if err := o.history.MarkDirty(ctx, seed.Version); err != nil {
		return err
	}

	if err := o.executor.Execute(ctx, seed, direction); err != nil {
		return err
	}

	switch direction {
	case Up:
		if err := o.history.SetVersion(ctx, seed.Version); err != nil {
			return err
		}

	case Down:
		if err := o.history.Remove(ctx, seed.Version); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported direction: %s", direction)
	}

	return nil
}

func (o *Orchestrator) findSeed(version uint64) (Seed, bool) {
	for i := len(o.seeds) - 1; i >= 0; i-- {
		if uint64(o.seeds[i].Version) == version {
			return o.seeds[i], true
		}
	}

	return Seed{}, false
}
