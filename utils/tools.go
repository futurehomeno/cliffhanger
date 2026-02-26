package utils

import (
	"regexp"
	"runtime/debug"
	"strings"

	log "github.com/sirupsen/logrus"
)

func FilterGoroutinesByKeywords(input string, keywords []string) string {
	var keywordsStr strings.Builder

	for _, k := range keywords {
		keywordsStr.WriteString(k + "|")
	}
	keywordRE := regexp.MustCompile(`(?i)\b(` + strings.Trim(keywordsStr.String(), "|") + `)\b`)

	lines := strings.Split(input, "\n")

	var (
		block    []string
		matched  bool
		out      []string
		inGorout bool
	)

	flush := func() {
		if inGorout && matched && len(block) > 0 {
			out = append(out, block...)
			out = append(out, "")
		}
		block = block[:0]
		matched = false
		inGorout = false
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "goroutine ") {
			flush()
			inGorout = true
			block = append(block, line)
			continue
		}

		if !inGorout {
			continue
		}

		block = append(block, line)

		if keywordRE.MatchString(line) {
			matched = true
		}
	}

	flush()

	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	return strings.Join(out, "\n")
}

func PrintStackOnRecover(name string, terminate bool) {
	if r := recover(); r != nil {
		log.Errorf("panic in %s:\n%s", name, string(debug.Stack()))
		if terminate {
			panic(r)
		} else {
			log.Error(r)
		}
	}
}
