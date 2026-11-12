package main

import "os"

func main() {
	if err := Execute(); err != nil {
		code, msg := exitCode(err)
		if msg != "" {
			_, _ = os.Stderr.WriteString(msg + "\n")
		}
		os.Exit(code)
	}
}
