package bootstrap_test

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/bootstrap"
)

func TestGetConfigurationDirectory(t *testing.T) { //nolint:paralleltest
	assert.Equal(t, "./", bootstrap.GetConfigurationDirectory())

	os.Args = append(os.Args, "-c=/my/configuration/directory")

	flag.Parse()

	assert.Equal(t, "/my/configuration/directory", bootstrap.GetConfigurationDirectory())
}

func TestGetWorkingDirectory(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()

	assert.NoError(t, err)
	assert.Equal(t, wd, bootstrap.GetWorkingDirectory())
}
