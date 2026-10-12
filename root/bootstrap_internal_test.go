package root

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogBootstrapDirs_LeavesUnparsedFlagsAlone(t *testing.T) { //nolint:paralleltest
	saved := flag.CommandLine
	t.Cleanup(func() { flag.CommandLine = saved })

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	logBootstrapDirs()

	assert.False(t, flag.Parsed())
	assert.Nil(t, flag.Lookup("c"))
}
