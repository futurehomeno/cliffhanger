package utils

import (
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func FilterGoroutinesByKeywords(input string, keywords []string) string {
	keywordsStr := ""

	for _, k := range keywords {
		keywordsStr += k + "|"
	}
	keywordRE := regexp.MustCompile(`(?i)\b(` + strings.Trim(keywordsStr, "|") + `)\b`)

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

func PrintStackOnRecover(reboot bool, msg string) {
	if r := recover(); r != nil {
		if msg != "" {
			log.Error(r)
		}
		log.Error(string(debug.Stack()))

		if reboot {
			time.Sleep(10 * time.Second)
			panic(r)
		} else {
			log.Error(r)
		}
	}
}
