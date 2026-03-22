package main

import (
	"fmt"
	"os"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[0;33m"
	colorBlue   = "\033[0;34m"
	colorCyan   = "\033[0;36m"
)

func logInfo(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorGreen+"[INFO]"+colorReset+" "+format+"\n", args...)
}

func logWarn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorYellow+"[WARN]"+colorReset+" "+format+"\n", args...)
}

func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorRed+"[ERROR]"+colorReset+" "+format+"\n", args...)
}

func logDry(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorCyan+"[DRY-RUN]"+colorReset+" "+format+"\n", args...)
}

func logVerbose(verbose bool, format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, colorBlue+"[DEBUG]"+colorReset+" "+format+"\n", args...)
	}
}
