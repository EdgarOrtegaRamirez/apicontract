// Package cli provides the command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/differ"
	"github.com/EdgarOrtegaRamirez/apicontract/internal/generator"
	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
	"github.com/EdgarOrtegaRamirez/apicontract/internal/validator"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apicontract",
	Short: "OpenAPI/Swagger API Contract Validation Toolkit",
	Long: `apicontract is a comprehensive CLI toolkit for validating, diffing,
and generating code from OpenAPI/Swagger specifications.`,
}

var validateCmd = &cobra.Command{
	Use:   "validate <spec-file>",
	Short: "Validate an OpenAPI specification",
	Long:  "Validate an OpenAPI/Swagger specification for structural correctness.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		spec, err := parser.ParseSpec(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := spec.ParsePaths(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing paths: %v\n", err)
			os.Exit(1)
		}

		v := validator.NewValidator(spec)
		issues := v.ValidateSpec()
		if len(issues) > 0 {
			fmt.Println("Specification issues:")
			for _, issue := range issues {
				fmt.Printf("  [%s] %s: %s\n", issue.Severity, issue.Category, issue.Message)
			}
			os.Exit(1)
		}

		endpoints := spec.GetEndpoints()
		fmt.Printf("✓ Spec is valid\n")
		fmt.Printf("  Title: %s v%s\n", spec.Info.Title, spec.Info.Version)
		fmt.Printf("  Endpoints: %d\n", len(endpoints))
		fmt.Printf("  Servers: %d\n", len(spec.Servers))
	},
}

var checkCmd = &cobra.Command{
	Use:   "check <spec-file> <base-url>",
	Short: "Check endpoints against a running API",
	Long:  "Send requests to a running API and validate responses against the spec.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		spec, err := parser.ParseSpec(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := spec.ParsePaths(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing paths: %v\n", err)
			os.Exit(1)
		}

		v := validator.NewValidator(spec)
		endpoints := spec.GetEndpoints()

		if len(endpoints) == 0 {
			fmt.Println("No endpoints to check")
			return
		}

		passed := 0
		failed := 0

		for _, ep := range endpoints {
			result, err := v.ValidateEndpoint(ep, args[1])
			if err != nil {
				fmt.Printf("  ✗ %s: %v\n", ep.String(), err)
				failed++
				continue
			}

			if result.Valid {
				fmt.Printf("  ✓ %s (%d) in %v\n", ep.String(), result.Status, result.Duration.Round(1000000))
				passed++
			} else {
				fmt.Printf("  ✗ %s (%d) — %d issue(s)\n", ep.String(), result.Status, len(result.Issues))
				for _, issue := range result.Issues {
					fmt.Printf("    [%s] %s: %s\n", issue.Severity, issue.Category, issue.Message)
				}
				failed++
			}
		}

		fmt.Printf("\n%d passed, %d failed out of %d endpoints\n", passed, failed, len(endpoints))
		if failed > 0 {
			os.Exit(1)
		}
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <old-spec> <new-spec>",
	Short: "Diff two OpenAPI specifications",
	Long:  "Compare two OpenAPI specs and report added, removed, and modified endpoints.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldSpec, err := parser.ParseSpec(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading old spec: %v\n", err)
			os.Exit(1)
		}
		if err := oldSpec.ParsePaths(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing old spec: %v\n", err)
			os.Exit(1)
		}

		newSpec, err := parser.ParseSpec(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading new spec: %v\n", err)
			os.Exit(1)
		}
		if err := newSpec.ParsePaths(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing new spec: %v\n", err)
			os.Exit(1)
		}

		result := differ.Diff(oldSpec, newSpec)
		fmt.Printf("API Diff: %s\n\n", result.Summary)

		if len(result.Changes) == 0 {
			fmt.Println("No changes detected.")
			return
		}

		for _, c := range result.Changes {
			symbol := " "
			switch c.Type {
			case differ.ChangeAdded:
				symbol = "+"
			case differ.ChangeRemoved:
				symbol = "-"
			case differ.ChangeModified:
				symbol = "~"
			case differ.ChangeBreaking:
				symbol = "!"
			}
			fmt.Printf("  [%s] %s %s — %s\n", symbol, c.Method, c.Path, c.Detail)
		}
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate <spec-file> --format <go|python|typescript>",
	Short: "Generate client code from an OpenAPI spec",
	Long:  "Generate client stub code in Go, Python, or TypeScript from an OpenAPI specification.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")
		spec, err := parser.ParseSpec(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := spec.ParsePaths(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing paths: %v\n", err)
			os.Exit(1)
		}

		code, err := generator.Generate(spec, generator.Format(format))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(code)
	},
}

var infoCmd = &cobra.Command{
	Use:   "info <spec-file>",
	Short: "Show spec information",
	Long:  "Display metadata and statistics about an OpenAPI specification.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		spec, err := parser.ParseSpec(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := spec.ParsePaths(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing paths: %v\n", err)
			os.Exit(1)
		}

		endpoints := spec.GetEndpoints()

		fmt.Printf("Title: %s\n", spec.Info.Title)
		fmt.Printf("Version: %s\n", spec.Info.Version)
		if spec.Info.Description != "" {
			fmt.Printf("Description: %s\n", spec.Info.Description)
		}
		fmt.Printf("Endpoints: %d\n", len(endpoints))
		fmt.Printf("Servers: %d\n", len(spec.Servers))

		// Count by method.
		methods := make(map[string]int)
		for _, ep := range endpoints {
			methods[ep.Method]++
		}
		fmt.Println("Methods:")
		for method, count := range methods {
			fmt.Printf("  %s: %d\n", method, count)
		}

		// Count by response code.
		respCodes := make(map[string]int)
		for _, ep := range endpoints {
			for code := range ep.Operation.Responses {
				respCodes[code]++
			}
		}
		if len(respCodes) > 0 {
			fmt.Println("Response codes:")
			for code, count := range respCodes {
				fmt.Printf("  %s: %d endpoints\n", code, count)
			}
		}
	},
}

func Execute() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(infoCmd)

	generateCmd.Flags().StringP("format", "f", "go", "Output format (go, python, typescript)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
