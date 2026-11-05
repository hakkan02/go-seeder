package seeder

import (
    "context"
    "database/sql"
    "fmt"
)

type History struct {
    db *sql.DB
}

// Initialize creates the seed_migrations table.
func (h *History) Initialize(ctx context.Context) error {
    query := `
        CREATE TABLE IF NOT EXISTS seed_migrations (
            version BIGINT PRIMARY KEY,
            dirty BOOLEAN NOT NULL DEFAULT FALSE,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `

    if _, err := h.db.ExecContext(ctx, query); err != nil {
        return fmt.Errorf("failed to initialize seed migrations: %w", err)
    }

    return nil
}

// Current returns the current migration state.
func (h *History) Current(ctx context.Context) (State, error) {
    query := `
        SELECT version, dirty
        FROM seed_migrations
        ORDER BY version DESC
        LIMIT 1
    `

    var state State

    err := h.db.QueryRowContext(ctx, query).Scan(
        &state.Version,
        &state.Dirty,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return State{}, nil
        }

        return State{}, fmt.Errorf(
            "failed to get current migration state: %w",
            err,
        )
    }

    return state, nil
}

// MarkDirty records the version currently being executed.
func (h *History) MarkDirty(ctx context.Context, version uint) error {
    query := `
        INSERT INTO seed_migrations (version, dirty)
        VALUES ($1, TRUE)
        ON CONFLICT (version)
        DO UPDATE SET dirty = TRUE, applied_at = NOW()
    `

    if _, err := h.db.ExecContext(ctx, query, version); err != nil {
        return fmt.Errorf("failed to mark migration as dirty: %w", err)
    }

    return nil
}

// SetVersion marks a migration as successfully applied.
func (h *History) SetVersion(ctx context.Context, version uint) error {
    query := `
        UPDATE seed_migrations
        SET dirty = FALSE,
            applied_at = NOW()
        WHERE version = $1
    `

    result, err := h.db.ExecContext(ctx, query, version)
    if err != nil {
        return fmt.Errorf("failed to set migration version: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to check affected rows: %w", err)
    }

    if rows == 0 {
        return fmt.Errorf("migration version %d not found", version)
    }

    return nil
}

// Remove removes a migration version from history.
func (h *History) Remove(ctx context.Context, version uint) error {
    query := `
        DELETE FROM seed_migrations
        WHERE version = $1
    `

    result, err := h.db.ExecContext(ctx, query, version)
    if err != nil {
        return fmt.Errorf("failed to remove migration version: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to check affected rows: %w", err)
    }

    if rows == 0 {
        return fmt.Errorf("migration version %d not found", version)
    }

    return nil
}