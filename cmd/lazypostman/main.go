package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitss1/lazypostman/internal/collection"
	"github.com/mitss1/lazypostman/internal/environment"
	"github.com/mitss1/lazypostman/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("lazypostman %s\n", version)
		os.Exit(0)
	}

	envMgr := environment.NewManager()

	var col *collection.Collection

	if len(os.Args) >= 2 && os.Args[1] != "--help" && os.Args[1] != "-h" {
		// Load collection from file
		colPath := os.Args[1]
		var err error
		col, err = collection.Load(colPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading collection: %s\n", err)
			os.Exit(1)
		}

		// Load environments from args
		for _, arg := range os.Args[2:] {
			if err := envMgr.LoadEnvironment(arg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load environment %s: %s\n", arg, err)
			}
		}

		// Auto-discover environment files in same directory
		colDir := filepath.Dir(colPath)
		if entries, err := os.ReadDir(colDir); err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasSuffix(name, ".postman_environment.json") {
					envPath := filepath.Join(colDir, name)
					alreadyLoaded := false
					for _, arg := range os.Args[2:] {
						abs1, _ := filepath.Abs(arg)
						abs2, _ := filepath.Abs(envPath)
						if abs1 == abs2 {
							alreadyLoaded = true
							break
						}
					}
					if !alreadyLoaded {
						_ = envMgr.LoadEnvironment(envPath)
					}
				}
			}
		}
	} else if len(os.Args) >= 2 {
		// --help flag
		printUsage()
		os.Exit(0)
	}

	// If no collection file, start with an empty one (user can browse from Postman Cloud)
	if col == nil {
		col = &collection.Collection{
			Info: collection.Info{
				Name:        "LazyPostman",
				Description: "Press 'o' to browse collections from Postman Cloud, or 'L' to login",
			},
		}
	}

	app := ui.NewApp(col, envMgr)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("LazyPostman - A terminal UI for Postman collections")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lazypostman                                    # Start empty, browse from Postman Cloud")
	fmt.Println("  lazypostman <collection.json>                  # Open a local collection file")
	fmt.Println("  lazypostman <collection.json> [env.json...]    # With environment files")
	fmt.Println()
	fmt.Println("Keybindings:")
	fmt.Println("  o          Browse collections from Postman Cloud")
	fmt.Println("  E          Browse environments from Postman Cloud")
	fmt.Println("  L          Login with Postman API key")
	fmt.Println("  e          Cycle local environments")
	fmt.Println("  i          Edit selected field (URL/param/header/body)")
	fmt.Println("  v          Open environment variables panel")
	fmt.Println("  j/k        Navigate / scroll")
	fmt.Println("  Enter      Send request")
	fmt.Println("  Tab        Switch panel")
	fmt.Println("  t          Switch tab (params/headers/body)")
	fmt.Println("  +/-        Maximize / restore panel")
	fmt.Println("  ?          Help")
	fmt.Println("  q          Quit")
}
