package seeder

import (
    "crypto/sha256"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
)

var seedPattern = regexp.MustCompile(
    `^(\d+)_(.+)\.(up|down)\.sql$`,
)

func Discover(directory string) ([]Seed, error) {
    entries, err := os.ReadDir(directory)
    if err != nil {
        return nil, fmt.Errorf("failed to read seed directory: %w", err)
    }

    seeds := make(map[uint]*Seed)

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        filename := entry.Name()

        matches := seedPattern.FindStringSubmatch(filename)
        if matches == nil {
            if strings.HasSuffix(filename, ".sql") {
                return nil, fmt.Errorf("invalid seed filename: %s", filename)
            }
            continue
        }

        versionValue, err := strconv.ParseUint(matches[1], 10, strconv.IntSize)
        if err != nil {
            return nil, fmt.Errorf("invalid seed version in %s: %w", filename, err)
        }

        version := uint(versionValue)
        identifier := matches[2]
        direction := Direction(matches[3])

        if version == 0 {
            return nil, fmt.Errorf("seed version cannot be zero: %s", filename)
        }

        seed, exists := seeds[version]
        if !exists {
            seed = &Seed{
                Version:    version,
                Identifier: identifier,
            }
            seeds[version] = seed
        } else if seed.Identifier != identifier {
            return nil, fmt.Errorf(
                "conflicting identifiers for seed version %d",
                version,
            )
        }

        content, err := os.ReadFile(filepath.Join(directory, filename))
        if err != nil {
            return nil, fmt.Errorf("failed to read seed file %s: %w", filename, err)
        }

        sql := string(content)

        if strings.TrimSpace(sql) == "" {
            return nil, fmt.Errorf("seed file cannot be empty: %s", filename)
        }

        switch direction {
        case Up:
            if seed.Up != "" {
                return nil, fmt.Errorf("duplicate UP seed for version %d", version)
            }
            seed.Up = sql

        case Down:
            if seed.Down != "" {
                return nil, fmt.Errorf("duplicate DOWN seed for version %d", version)
            }
            seed.Down = sql
        }
    }

    result := make([]Seed, 0, len(seeds))

    for _, seed := range seeds {
        if seed.Up == "" || seed.Down == "" {
            return nil, fmt.Errorf(
                "incomplete seed version %d: UP and DOWN files are required",
                seed.Version,
            )
        }

        seed.Checksum = calculateChecksum(seed.Up, seed.Down)
        result = append(result, *seed)
    }

    sort.Slice(result, func(i, j int) bool {
        return result[i].Version < result[j].Version
    })

    return result, nil
}

func calculateChecksum(up, down string) string {
    content := up + "\x00" + down
    checksum := sha256.Sum256([]byte(content))

    return fmt.Sprintf("%x", checksum)
}