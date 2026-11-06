package seeder

import (
	"context"
	"database/sql"
	"fmt"
)

// History manages the state of applied seeds in the database.
type History struct {
	db *sql.DB
}

// NewHistory creates and returns a new History instance.
func NewHistory(db *sql.DB) *History {
	return &History{db: db}
}

// Initialize creates the seed_migrations table if it does not exist.
func (h *History) Initialize(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS seed_migrations (
			version BIGINT PRIMARY KEY,
			checksum VARCHAR(64) NOT NULL DEFAULT '',
			dirty BOOLEAN NOT NULL DEFAULT FALSE,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	if _, err := h.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("initialize seed history: %w", err)
	}
	return nil
}

// Current returns the current state of the seed history.
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

	if err == sql.ErrNoRows {
		return State{}, nil
	}

	if err != nil {
		return State{}, fmt.Errorf(
			"get current seed state: %w", err,
		)
	}
	return state, nil
}

// MarkDirty marks the given seed as dirty in the seed history.
func (h *History) MarkDirty(ctx context.Context, seed Seed) error {
	query := `
		INSERT INTO seed_migrations (
			version, checksum, dirty
		)
		VALUES ($1, $2, TRUE)
		ON CONFLICT (version)
		DO UPDATE SET
			checksum = EXCLUDED.checksum,
			dirty = TRUE,
			applied_at = NOW()
	`

	if _, err := h.db.ExecContext(
		ctx,
		query,
		seed.Version,
		seed.Checksum,
	); err != nil {
		return fmt.Errorf("mark seed dirty: %w", err)
	}
	return nil
}

// FinalizeTx updates the history in the same transaction
// as the seed SQL execution.
func (h *History) FinalizeTx(ctx context.Context, tx *sql.Tx,	seed Seed, direction Direction) error {
	var query string
	var args []any

	switch direction {
	case Up:
		query = `
			UPDATE seed_migrations
			SET dirty = FALSE,
			    checksum = $2,
			    applied_at = NOW()
			WHERE version = $1
		`
		args = []any{seed.Version, seed.Checksum}

	case Down:
		query = `
			DELETE FROM seed_migrations
			WHERE version = $1
		`
		args = []any{seed.Version}

	default:
		return fmt.Errorf("unsupported direction: %s", direction)
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("finalize seed history: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}

	if rows != 1 {
		return fmt.Errorf(
			"expected one history record for seed %d, got %d",
			seed.Version,
			rows,
		)
	}
	return nil
}

// VerifyChecksums checks that the checksums of applied seeds match the discovered seeds.
func (h *History) VerifyChecksums(ctx context.Context, seeds []Seed) error {
    rows, err := h.db.QueryContext(
        ctx,
        `SELECT version, checksum FROM seed_migrations`,
    )
    if err != nil {
        return fmt.Errorf("read seed checksums: %w", err)
    }
    defer rows.Close()

    checksums := make(map[uint64]string)

    for rows.Next() {
        var version uint64
        var checksum string

        if err := rows.Scan(&version, &checksum); err != nil {
            return fmt.Errorf("scan seed checksum: %w", err)
        }

        checksums[version] = checksum
    }

    if err := rows.Err(); err != nil {
        return fmt.Errorf("iterate seed checksums: %w", err)
    }

    discovered := make(map[uint64]string, len(seeds))

    for _, seed := range seeds {
        discovered[seed.Version] = seed.Checksum
    }

    for version, checksum := range checksums {
        current, exists := discovered[version]
        if !exists {
            return fmt.Errorf(
                "applied seed %d is missing from source",
                version,
            )
        }

        if checksum != "" && checksum != current {
            return fmt.Errorf(
                "checksum mismatch for seed %d",
                version,
            )
        }
    }
    return nil
}