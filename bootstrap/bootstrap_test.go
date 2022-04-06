package bootstrap_test

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/bootstrap"
)

func TestGetConfigurationDirectory(t *testing.T) { //nolint:paralleltest
	os.Args = append(os.Args, "-c=/my/configuration/directory")

	assert.Equal(t, "/my/configuration/directory", bootstrap.GetConfigurationDirectory())
}

func TestGetWorkingDirectory(t *testing.T) { //nolint:paralleltest
	wd, err := os.Getwd()

	assert.NoError(t, err)
	assert.Equal(t, wd, bootstrap.GetWorkingDirectory())

	os.Args = append(os.Args, "-w=/my/working/directory")

	flag.Parse()

	assert.Equal(t, "/my/working/directory", bootstrap.GetWorkingDirectory())
}
