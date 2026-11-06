package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"seeder/seeder"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	source := flag.String(
		"source",
		"",
		"seed directory",
	)

	database := flag.String(
		"database",
		"",
		"database connection string",
	)

	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		return fmt.Errorf(
			"expected command: up, down, reset or version",
		)
	}

	if *source == "" || *database == "" {
		return fmt.Errorf(
			"source and database are required",
		)
	}

	db, err := sql.Open("pgx", *database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	s, err := seeder.New(ctx, seeder.Config{
		DB: db,
		FS: os.DirFS(*source),
	})
	if err != nil {
		return fmt.Errorf("initialize seeder: %w", err)
	}

	var steps int

	switch args[0] {
	case "up", "down":
		steps = 1

		if len(args) > 2 {
			return fmt.Errorf("too many arguments")
		}

		if len(args) == 2 {
			if _, err := fmt.Sscanf(args[1], "%d", &steps); err != nil || steps <= 0 {
				return fmt.Errorf("steps must be positive")
			}
		}

		if args[0] == "up" {
			return s.Up(ctx, steps)
		}

		return s.Down(ctx, steps)

	case "reset":
		if len(args) != 1 {
			return fmt.Errorf("reset accepts no arguments")
		}
		return s.Reset(ctx)

	case "version":
		if len(args) != 1 {
			return fmt.Errorf("version accepts no arguments")
		}

		state, err := s.Version(ctx)
		if err != nil {
			return err
		}

		fmt.Printf(
			"Version: %d | Dirty: %t\n",
			state.Version,
			state.Dirty,
		)

		return nil

	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
