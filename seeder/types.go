package seeder

// Direction is either up or down.
type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
)

// Seed represents a database seed.
type Seed struct {
	Version    uint64
	Identifier string
	Up         string
	Down       string
	Checksum   string
}


// State represents the current state of the database.
type State struct {
    Version uint64
    Dirty   bool
}