package seeder

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var seedPattern = regexp.MustCompile(
	`^(\d+)_(.+)\.(up|down)\.sql$`,
)

// DiscoverFS discovers and returns all seed files from the given filesystem.
// It returns a slice of Seed structs and an error if any issues are encountered during discovery.
// The seed files must follow the naming convention: <version>_<identifier>.(up|down).sql
func DiscoverFS(fsys fs.FS) ([]Seed, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read seed directory: %w", err)
	}

	seeds := make(map[uint64]*Seed)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		matches := seedPattern.FindStringSubmatch(filename)
		if matches == nil {
			if strings.HasSuffix(filename, ".sql") {
				return nil, fmt.Errorf(
					"invalid seed filename: %s", filename,
				)
			}
			continue
		}

		version, err := strconv.ParseUint(matches[1], 10, 63)
		if err != nil || version == 0 {
			return nil, fmt.Errorf(
				"invalid seed version in %s", filename,
			)
		}

		identifier := matches[2]
		direction := Direction(matches[3])

		seed, exists := seeds[version]

		if !exists {
			seed = &Seed{
				Version:    version,
				Identifier: identifier,
			}
			seeds[version] = seed
		} else if seed.Identifier != identifier {
			return nil, fmt.Errorf(
				"conflicting identifiers for version %d",
				version,
			)
		}

		content, err := fs.ReadFile(
			fsys,
			path.Clean(filename),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"read seed %s: %w", filename, err,
			)
		}

		query := string(content)

		if strings.TrimSpace(query) == "" {
			return nil, fmt.Errorf(
				"seed file cannot be empty: %s", filename,
			)
		}

		switch direction {
		case Up:
			if seed.Up != "" {
				return nil, fmt.Errorf(
					"duplicate UP seed for version %d", version,
				)
			}
			seed.Up = query

		case Down:
			if seed.Down != "" {
				return nil, fmt.Errorf(
					"duplicate DOWN seed for version %d", version,
				)
			}
			seed.Down = query
		}
	}

	result := make([]Seed, 0, len(seeds))

	for _, seed := range seeds {
		if seed.Up == "" || seed.Down == "" {
			return nil, fmt.Errorf(
				"incomplete seed version %d: UP and DOWN required",
				seed.Version,
			)
		}

		seed.Checksum = calculateChecksum(
			seed.Up,
			seed.Down,
		)

		result = append(result, *seed)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

// calculateChecksum computes a SHA-256 checksum for the given UP and DOWN seed queries.
func calculateChecksum(up, down string) string {
	content := up + "\x00" + down
	checksum := sha256.Sum256([]byte(content))

	return fmt.Sprintf("%x", checksum)
}
