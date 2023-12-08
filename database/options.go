package database

// config is the configuration for the database.
type config struct {
	workdir              string
	filename             string
	compactionSize       int
	compactionPercentage int
}

// newConfig creates a new configuration object.
func newConfig() *config {
	return &config{}
}

// withDefaults sets the default values for the configuration.
func (c *config) withDefaults() *config {
	c.filename = "data"
	c.compactionSize = 2 * 1024 * 1024

	return c
}

// apply applies the provided options to the configuration.
func (c *config) apply(opts ...Option) *config {
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Option is a configuration option.
type Option func(*config)

// WithWorkdir sets the workdir for the database.
func WithWorkdir(workdir string) Option {
	return func(c *config) {
		c.workdir = workdir
	}
}

// WithFilename sets the filename for the database.
func WithFilename(filename string) Option {
	return func(c *config) {
		c.filename = filename
	}
}

// WithCompactionSize sets the compaction size for the database.
func WithCompactionSize(size int) Option {
	return func(c *config) {
		c.compactionSize = size
	}
}

// WithCompactionPercentage sets the compaction percentage for the database.
func WithCompactionPercentage(percentage int) Option {
	return func(c *config) {
		c.compactionPercentage = percentage
	}
}
