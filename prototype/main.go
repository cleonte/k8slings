package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const version = "0.1.0"

const usage = `k8slings - small exercises to get you fluent in Kubernetes (runs on a local kind cluster)

Usage:
  k8slings init [dir]     create a workspace (default ./k8slings-exercises) and the kind cluster
  k8slings watch          run the current exercise and re-check whenever you save (default)
  k8slings run [id]       check one exercise once (default: the current one)
  k8slings list           show all exercises and your progress
  k8slings hint [id]      show the hint for an exercise
  k8slings reset [id]     restore an exercise file to its original (broken) state
  k8slings verify         check every exercise against the files in your workspace
  k8slings destroy        delete the kind cluster (everything is gone, progress file stays)
  k8slings version
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, red("error: ")+err.Error())
		os.Exit(1)
	}
}

func pick(ws *Workspace, args []string) (*Exercise, error) {
	if len(args) > 0 {
		if ex := findExercise(args[0]); ex != nil {
			return ex, nil
		}
		return nil, fmt.Errorf("unknown exercise %q (try `k8slings list`)", args[0])
	}
	if ex := ws.next(); ex != nil {
		return ex, nil
	}
	return nil, fmt.Errorf("all exercises are done")
}

func run(args []string) error {
	cmd := "watch"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "version", "--version":
		fmt.Println("k8slings", version)
		return nil
	case "init":
		return cmdInit(args)
	case "destroy":
		return deleteCluster()
	}

	ws, err := findWorkspace()
	if err != nil {
		return err
	}
	switch cmd {
	case "watch":
		return watch(ws)
	case "list":
		printList(ws)
		return nil
	case "hint":
		ex, err := pick(ws, args)
		if err != nil {
			return err
		}
		fmt.Println(yellow("Hint for "+ex.ID+":") + "\n" + readEmbedded(ex, "hint.md"))
		return nil
	case "reset":
		ex, err := pick(ws, args)
		if err != nil {
			return err
		}
		delete(ws.Progress, ex.ID)
		if err := ws.extract(ex.ID, true); err != nil {
			return err
		}
		fmt.Println("Reset", ex.ID)
		return ws.save()
	case "run":
		ex, err := pick(ws, args)
		if err != nil {
			return err
		}
		if err := Run(ws, ex, func(s string) { fmt.Println(dim("  " + s)) }); err != nil {
			fmt.Println(red("✗ " + ex.ID))
			return err
		}
		fmt.Println(green("✓ Passed: " + ex.ID))
		ws.Progress[ex.ID] = true
		return ws.save()
	case "verify":
		failed := 0
		for _, ex := range registry {
			if err := Run(ws, ex, func(string) {}); err != nil {
				failed++
				fmt.Printf("%s %s: %s\n", red("✗"), ex.ID, firstLine(err.Error()))
				continue
			}
			ws.Progress[ex.ID] = true
			fmt.Printf("%s %s\n", green("✓"), ex.ID)
		}
		ws.save()
		if failed > 0 {
			return fmt.Errorf("%d exercise(s) not passing", failed)
		}
		return nil
	default:
		fmt.Print(usage)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func cmdInit(args []string) error {
	dir := "k8slings-exercises"
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	ws := &Workspace{Root: abs, Progress: map[string]bool{}}
	if existing, err := os.ReadFile(filepath.Join(abs, wsFile)); err == nil && len(existing) > 0 {
		os.Chdir(abs)
		if w, err := findWorkspace(); err == nil {
			ws = w
		}
	}
	if err := ws.extract("", false); err != nil {
		return err
	}
	if err := ws.save(); err != nil {
		return err
	}
	if err := createCluster(); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(green("Ready!") + " Next:")
	fmt.Printf("  cd %s\n  k8slings watch\n", dir)
	return nil
}
