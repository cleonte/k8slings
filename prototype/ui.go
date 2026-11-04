package main

import (
	"fmt"
	"os"
)

var useColor = os.Getenv("NO_COLOR") == ""

func c(code, s string) string {
	if !useColor {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func green(s string) string  { return c("32", s) }
func red(s string) string    { return c("31", s) }
func yellow(s string) string { return c("33", s) }
func cyan(s string) string   { return c("36", s) }
func bold(s string) string   { return c("1", s) }
func dim(s string) string    { return c("2", s) }

func clearScreen() {
	if useColor {
		fmt.Print("\x1b[2J\x1b[H")
	}
}

func progressBar(done, total, width int) string {
	if total == 0 {
		return ""
	}
	filled := done * width / total
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "#"
		} else {
			bar += "-"
		}
	}
	return fmt.Sprintf("[%s] %d/%d", bar, done, total)
}
