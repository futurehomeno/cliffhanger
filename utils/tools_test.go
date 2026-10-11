package utils_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/utils"
)

const goroutineDump = `goroutine 1 [running]:
main.main()
	/app/main.go:10

goroutine 7 [semacquire]:
sync.(*Mutex).Lock(...)
	/usr/lib/go/src/sync/mutex.go:90
github.com/x/edge.(*client).poll()

goroutine 9 [mutex wait]:
net/http.(*conn).serve()

goroutine 12 [select]:
github.com/x/edge.(*stream).run()
`

func TestFilterGoroutinesByKeywords(t *testing.T) {
	t.Parallel()

	out := utils.FilterGoroutinesByKeywords(goroutineDump, []string{"MUTEX"})
	assert.Equal(t, "goroutine 7 [semacquire]:\nsync.(*Mutex).Lock(...)\n\t/usr/lib/go/src/sync/mutex.go:90\ngithub.com/x/edge.(*client).poll()", out,
		"only the block mentioning the keyword below its header is kept, intact and without trailing blank lines")

	out = utils.FilterGoroutinesByKeywords(goroutineDump, []string{"stream", "client"})
	assert.Contains(t, out, "goroutine 7 ")
	assert.Contains(t, out, "\n\ngoroutine 12 [select]:\ngithub.com/x/edge.(*stream).run()")
	assert.NotContains(t, out, "goroutine 1 ")
	assert.NotContains(t, out, "goroutine 9 ")

	assert.Empty(t, utils.FilterGoroutinesByKeywords(goroutineDump, []string{"wait"}), "a keyword only in a header line keeps nothing")
	assert.Empty(t, utils.FilterGoroutinesByKeywords(goroutineDump, []string{"mute"}), "keywords match whole words only")
}

func TestPrintStackOnRecover(t *testing.T) { //nolint:paralleltest
	hook := test.NewGlobal()
	defer log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	assert.NotPanics(t, func() {
		defer utils.PrintStackOnRecover("edge", false)

		panic("boom")
	})

	entry := hook.LastEntry()
	if assert.NotNil(t, entry) {
		assert.Equal(t, log.ErrorLevel, entry.Level)
		assert.Contains(t, entry.Message, "[edge] Panic: boom")
		assert.Contains(t, entry.Message, "goroutine ")
	}

	assert.PanicsWithValue(t, "boom", func() {
		defer utils.PrintStackOnRecover("edge", true)

		panic("boom")
	})

	hook.Reset()

	assert.NotPanics(t, func() {
		defer utils.PrintStackOnRecover("edge", true)
	})
	assert.Empty(t, hook.AllEntries(), "nothing is logged when nothing panicked")
}
