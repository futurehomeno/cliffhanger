package database

type config struct {
	workdir              string
	filename             string
	compactionSize       int
	compactionPercentage int
}

func newConfig() *config {
	return &config{}
}

func (c *config) withDefaults() *config {
	c.filename = "data"
	c.compactionSize = 2 * 1024 * 1024

	return c
}

func (c *config) apply(opts ...Option) *config {
	for _, opt := range opts {
		opt(c)
	}

	return c
}

type Option func(*config)

func WithWorkdir(workdir string) Option {
	return func(c *config) {
		c.workdir = workdir
	}
}

func WithFilename(filename string) Option {
	return func(c *config) {
		c.filename = filename
	}
}

func WithCompactionSize(size int) Option {
	return func(c *config) {
		c.compactionSize = size
	}
}

func WithCompactionPercentage(percentage int) Option {
	return func(c *config) {
		c.compactionPercentage = percentage
	}
}
