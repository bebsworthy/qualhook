// Package main provides man page generation for qualhook
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manDir string

// manCmd represents the man command
var manCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man pages for qualhook",
	Long: `Generate man pages for qualhook and all its subcommands.

Man pages provide detailed documentation accessible via the 'man' command
on Unix-like systems. They include comprehensive information about commands,
flags, examples, and exit codes.

The generated man pages follow the standard man page format and can be
installed system-wide for easy access.`,
	Example: `  # Generate man pages in the current directory
  qualhook man

  # Generate man pages in a specific directory
  qualhook man --dir ./docs/man

  # Generate and install man pages (requires sudo on most systems)
  qualhook man --dir /tmp/qualhook-man
  sudo cp /tmp/qualhook-man/* /usr/local/share/man/man1/
  sudo mandb  # Update man database

  # View installed man page
  man qualhook
  man qualhook-lint
  man qualhook-config`,
	RunE: runGenerateMan,
}

func init() {
	manCmd.Flags().StringVar(&manDir, "dir", ".", "Directory to write man pages to")
}

func runGenerateMan(cmd *cobra.Command, args []string) error {
	// Ensure directory exists
	if err := os.MkdirAll(manDir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Set up header for man pages
	header := &doc.GenManHeader{
		Title:   "QUALHOOK",
		Section: "1",
		Source:  fmt.Sprintf("qualhook %s", Version),
		Manual:  "Qualhook Manual",
	}

	// Generate man pages
	err := doc.GenManTree(cmd.Root(), header, manDir)
	if err != nil {
		return fmt.Errorf("failed to generate man pages: %w", err)
	}

	// List generated files
	fmt.Printf("✅ Man pages generated successfully in: %s\n\n", manDir)
	fmt.Println("Generated files:")

	files, err := filepath.Glob(filepath.Join(manDir, "*.1"))
	if err != nil {
		return fmt.Errorf("failed to list generated files: %w", err)
	}

	for _, file := range files {
		fmt.Printf("  • %s\n", filepath.Base(file))
	}

	fmt.Println("\nTo install man pages system-wide:")
	fmt.Printf("  sudo cp %s/*.1 /usr/local/share/man/man1/\n", manDir)
	fmt.Println("  sudo mandb")

	return nil
}
