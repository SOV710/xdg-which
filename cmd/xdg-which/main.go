package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sov710/xdg-which/internal/xdg"
)

func main() {
	var desktop string
	var quiet bool
	flag.StringVar(&desktop, "desktop", "", "desktop environment name used for OnlyShowIn/NotShowIn checks")
	flag.BoolVar(&quiet, "q", false, "print only the selected path")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	result, err := xdg.Lookup(flag.Arg(0), xdg.Options{Desktop: desktop})
	if err != nil {
		fmt.Fprintln(os.Stderr, "xdg-which:", err)
		os.Exit(2)
	}
	if len(result.Candidates) == 0 {
		if !quiet {
			fmt.Fprintf(os.Stderr, "%s: not found in XDG application directories\n", result.ID)
		}
		os.Exit(1)
	}

	selected := result.Candidates[0]
	if quiet {
		fmt.Println(selected.Entry.Path)
		if len(selected.Problems) > 0 {
			os.Exit(3)
		}
		return
	}

	fmt.Printf("id: %s\n", result.ID)
	fmt.Printf("selected: %s\n", selected.Entry.Path)
	if result.MatchMode == "trimmed" {
		fmt.Printf("matched id: %s\n", selected.Entry.ID)
		fmt.Println("match: trimmed reverse-domain prefix")
	}
	if len(selected.Problems) == 0 {
		fmt.Println("status: visible")
	} else {
		fmt.Printf("status: questionable (%s)\n", strings.Join(selected.Problems, "; "))
	}

	printField("name", selected.Entry.Keys["Name"])
	printField("exec", selected.Entry.Keys["Exec"])
	printField("mime types", selected.Entry.Keys["MimeType"])

	if len(result.Candidates) > 1 {
		fmt.Println()
		fmt.Println("other candidates:")
		for _, candidate := range result.Candidates[1:] {
			reasons := candidate.Problems
			if len(reasons) == 0 {
				reasons = []string{"lower priority"}
			}
			fmt.Printf("- %s (%s)\n", candidate.Entry.Path, strings.Join(reasons, "; "))
		}
	}

	if len(selected.Problems) > 0 {
		os.Exit(3)
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: xdg-which [options] <desktop-id|desktop-file>\n\n")
	flag.PrintDefaults()
}

func printField(label, value string) {
	if value != "" {
		fmt.Printf("%s: %s\n", label, value)
	}
}
