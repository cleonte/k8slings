package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func printList(ws *Workspace) {
	fmt.Println(bold("Exercises") + "  " + progressBar(ws.doneCount(), len(registry), 20))
	cur := ws.next()
	for _, ex := range registry {
		mark := "  "
		switch {
		case ws.Progress[ex.ID]:
			mark = green("✓ ")
		case ex == cur:
			mark = yellow("→ ")
		}
		fmt.Printf(" %s%-24s %s\n", mark, ex.ID, ex.Title)
	}
}

func printHeader(ws *Workspace, ex *Exercise) {
	clearScreen()
	fmt.Println(bold(cyan("k8slings")) + "  " + progressBar(ws.doneCount(), len(registry), 20))
	fmt.Println()
	fmt.Printf("%s %s\n", bold("Exercise:"), ex.ID+" - "+ex.Title)
	fmt.Printf("%s %s\n", bold("Edit:    "), relPath(ws, ws.exercisePath(ex)))
	fmt.Printf("%s %s\n\n", bold("Read:    "), relPath(ws, ws.exercisePath(ex))[:len(relPath(ws, ws.exercisePath(ex)))-len("exercise.yaml")]+"README.md")
}

func runOnce(ws *Workspace, ex *Exercise) bool {
	printHeader(ws, ex)
	err := Run(ws, ex, func(s string) { fmt.Println(dim("  " + s)) })
	fmt.Println()
	switch {
	case err == nil:
		fmt.Println(green("✓ Passed: " + ex.ID))
		ws.Progress[ex.ID] = true
		ws.save()
		return true
	case err == errNotDone:
		fmt.Println(yellow("• " + err.Error()))
	default:
		fmt.Println(red("✗ Not yet:"))
		fmt.Println(indent(err.Error()))
	}
	fmt.Println()
	fmt.Println(dim("h = hint · l = list · r = re-run · q = quit  (save the file to re-check automatically)"))
	return false
}

func watch(ws *Workspace) error {
	if err := requireTools("kubectl"); err != nil {
		return err
	}
	cmds := make(chan string)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			cmds <- strings.TrimSpace(sc.Text())
		}
		close(cmds)
	}()

	for {
		ex := ws.next()
		if ex == nil {
			clearScreen()
			fmt.Println(green(bold("🎉 All exercises complete!")))
			fmt.Println("Clean up with: k8slings destroy")
			return nil
		}
		path := ws.exercisePath(ex)
		lastMod := modTime(path)
		if runOnce(ws, ex) {
			time.Sleep(1500 * time.Millisecond)
			continue
		}
		// Wait for edits or commands.
	wait:
		for {
			select {
			case cmd, ok := <-cmds:
				if !ok {
					return nil
				}
				switch cmd {
				case "h", "hint":
					fmt.Println("\n" + yellow("Hint:") + "\n" + readEmbedded(ex, "hint.md"))
				case "l", "list":
					fmt.Println()
					printList(ws)
				case "r", "":
					if runOnce(ws, ex) {
						break wait
					}
					lastMod = modTime(path)
				case "q", "quit":
					return nil
				}
			case <-time.After(500 * time.Millisecond):
				if m := modTime(path); m != lastMod {
					lastMod = m
					if runOnce(ws, ex) {
						break wait
					}
					lastMod = modTime(path)
				}
			}
		}
		time.Sleep(1500 * time.Millisecond)
	}
}

func modTime(p string) time.Time {
	st, err := os.Stat(p)
	if err != nil {
		return time.Time{}
	}
	return st.ModTime()
}
