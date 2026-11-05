package seeder

// Direction is either up or down.
type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
)

type Seed struct {
	Version    uint
	Identifier string
	Up         string
	Down       string
	Checksum   string
}

type State struct {
    Version uint64
    Dirty   bool
}