package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"td-file/config"
	"td-file/parser"
	"td-file/sync"
	"td-file/tui"
)

func main() {
	var todoFileFlag string
	flag.StringVar(&todoFileFlag, "todo-file", "", "Path to todo file (overrides config)")
	flag.StringVar(&todoFileFlag, "f", "", "Path to todo file (shorthand, overrides config)")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var todoPath string
	if todoFileFlag != "" {
		todoPath = todoFileFlag
		fmt.Printf("Using todo file from flag: %s\n", todoPath)
	} else {
		todoPath, err = config.ResolveTodoPath(cfg)
		if err != nil {
			log.Fatalf("Failed to resolve todo path: %v", err)
		}
	}

	// Ensure the todo directory exists
	if err := os.MkdirAll(filepath.Dir(todoPath), 0755); err != nil {
		log.Fatalf("Failed to create todo directory: %v", err)
	}

	if _, err := os.Stat(todoPath); os.IsNotExist(err) {
		fmt.Printf("Todo file '%s' does not exist. Please create it and restart the app.\n", todoPath)
		os.Exit(1)
	}

	blocks, err := parser.ExtractTdBlocks(todoPath)
	if err != nil {
		fmt.Println("Error reading todo file:", err)
		os.Exit(1)
	}
	if len(blocks) == 0 {
		fmt.Println("No :td blocks found. No todos in scope.")
		return
	}

	todos := parser.ParseTodos(blocks)

	syncer := sync.NewFileSynchronizer(todoPath)
	if err := syncer.Start(); err != nil {
		fmt.Println("Error starting file synchronizer:", err)
		os.Exit(1)
	}
	defer syncer.Stop()

	maxID := 0
	for i := range todos {
		if todos[i].ID > maxID {
			maxID = todos[i].ID
		}
	}
	if err := tui.StartTUI(todos, syncer, maxID); err != nil {
		fmt.Println("Error running TUI:", err)
		os.Exit(1)
	}
}
