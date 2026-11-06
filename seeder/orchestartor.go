package seeder

import (
	"slices"
	"context"
	"fmt"
)

// Orchestrator manages the execution of seeds against the database.
type Orchestrator struct {
	seeds    []Seed
	history  *History
	executor *Executor
}

// NewOrchestrator creates and returns a new Orchestrator instance.
func NewOrchestrator(seeds []Seed, history *History, executor *Executor) *Orchestrator {
	return &Orchestrator{
		seeds:    seeds,
		history:  history,
		executor: executor,
	}
}

// Up executes pending seeds.
// steps == 0 means execute all pending seeds.
func (o *Orchestrator) Up(ctx context.Context, steps int) error {
	if steps < 0 {
		return fmt.Errorf("steps cannot be negative")
	}

	state, err := o.history.Current(ctx)
	if err != nil {
		return err
	}

	if state.Dirty {
		return fmt.Errorf(
			"database is dirty at version %d",
			state.Version,
		)
	}

	if err := o.verifyChecksums(ctx); err != nil {
		return err
	}

	applied := 0

	for _, seed := range o.seeds {
		if seed.Version <= state.Version {
			continue
		}

		if steps > 0 && applied >= steps {
			break
		}

		if err := o.apply(ctx, seed, Up); err != nil {
			return err
		}

		applied++
	}

	return nil
}

// Down rolls back the specified number of seeds.

func (o *Orchestrator) Down(ctx context.Context, steps int) error {
	if steps <= 0 {
		return fmt.Errorf(
			"rollback steps must be greater than zero",
		)
	}

	for range steps {
		state, err := o.history.Current(ctx)
		if err != nil {
			return err
		}

		if state.Dirty {
			return fmt.Errorf(
				"database is dirty at version %d",
				state.Version,
			)
		}

		if state.Version == 0 {
			return nil
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

		if state.Dirty {
			return fmt.Errorf(
				"database is dirty at version %d",
				state.Version,
			)
		}

		if err := o.Down(ctx, 1); err != nil {
			return err
		}
	}
}

// Version returns the current seed state.
func (o *Orchestrator) Version(
	ctx context.Context,
) (State, error) {
	return o.history.Current(ctx)
}

func (o *Orchestrator) apply(
	ctx context.Context,
	seed Seed,
	direction Direction,
) error {
	if err := o.history.MarkDirty(ctx, seed); err != nil {
		return err
	}

	tx, err := o.history.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin transaction for seed %d: %w",
			seed.Version,
			err,
		)
	}

	defer tx.Rollback()

	if err := o.executor.Execute(
		ctx,
		tx,
		seed,
		direction,
	); err != nil {
		return err
	}

	if err := o.history.FinalizeTx(
		ctx,
		tx,
		seed,
		direction,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit seed %d: %w",
			seed.Version,
			err,
		)
	}

	return nil
}

func (o *Orchestrator) findSeed(version uint64) (Seed, bool) {
	for _, v := range slices.Backward(o.seeds) {
		if v.Version == version {
			return v, true
		}
	}

	return Seed{}, false
}

func (o *Orchestrator) verifyChecksums(ctx context.Context) error {
	return o.history.VerifyChecksums(ctx, o.seeds)
}
