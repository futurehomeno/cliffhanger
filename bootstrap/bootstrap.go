package bootstrap

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
)

// GetConfigurationDirectory returns a configuration directory passed through the -c option with a fallback to a relative path.
func GetConfigurationDirectory() string {
	const c = "c"

	if flag.Lookup(c) == nil {
		flag.String(c, "", "Configuration directory.")
		flag.Parse()
	}

	dir := flag.Lookup(c).Value.String()
	if dir != "" {
		return dir
	}

	return "./"
}

func GetWorkingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "./"
	}

	return dir
}

func WaitForShutdown() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	<-signals
}
