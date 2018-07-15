package cli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

const (
	// GitCommitKey is the key for the program's GitCommit data.
	GitCommitKey ContextKey = "program.GitCommit"
	// VersionKey is the key for the program's Version data.
	VersionKey ContextKey = "program.Version"
)

// ContextKey defines the type for holding keys in the context.
type ContextKey string

// Program defines the struct for holding information about the program.
type Program struct {
	// Name of the program. Defaults to path.Base(os.Args[0]).
	Name string
	// Description of the program.
	Description string
	// Version of the program.
	Version string
	// GitCommit information for the program.
	GitCommit string

	// Commands in the program.
	Commands []Command
	// FlagSet holds the common/global flags for the program.
	FlagSet *flag.FlagSet

	// Before defines a function to execute before any subcommands are run,
	// but after the context is ready.
	// If a non-nil error is returned, no subcommands are run.
	Before func(context.Context) error
	// After defines a function to execute after any subcommands are run,
	// but after the subcommand has finished.
	// It is run even if the subcommand returns an error.
	After func(context.Context) error

	// Action is the function to execute when no subcommands are specified.
	Action func(context.Context) error
}

// Command defines the interface for each command in a program.
type Command interface {
	Name() string      // "foobar"
	Args() string      // "<baz> [quux...]"
	ShortHelp() string // "Foo the first bar"
	LongHelp() string  // "Foo the first bar meeting the following conditions..."

	// Hidden indicates whether the command should be hidden from the help output.
	Hidden() bool

	// Register command specific flags.
	Register(*flag.FlagSet)
	// Run executes the function for the command with a context and the command arguments.
	Run(context.Context, []string) error
}

// NewProgram creates a new Program with some reasonable defaults for Name,
// Description, and Version.
func NewProgram() *Program {
	return &Program{
		Name:        filepath.Base(os.Args[0]),
		Description: "A new command line program.",
		Version:     "0.0.0",
	}
}

// Run is the entry point for the program. It parses the arguments and executes
// the commands.
func (p *Program) Run() {
	// Create the context with the values we need to pass to the version command.
	ctx := context.WithValue(context.Background(), GitCommitKey, p.GitCommit)
	ctx = context.WithValue(ctx, VersionKey, p.Version)

	// Append the version command to the list of commands by default.
	p.Commands = append(p.Commands, &versionCommand{})

	// TODO(jessfraz): Find a better way to tell that they passed -h through as a flag.
	if len(os.Args) > 1 &&
		(strings.Contains(strings.ToLower(os.Args[1]), "help") ||
			strings.ToLower(os.Args[1]) == "-h") {
		p.usage(ctx)
		os.Exit(1)
	}

	// Set the default action to print the usage if it is undefined.
	if p.Action == nil {
		p.Action = p.usage
	}

	// If we are not running commands then automatically run the main action of
	// the program instead.
	if len(p.Commands) <= 1 {
		// Set the default flagset if our flagset is undefined.
		if p.FlagSet == nil {
			p.FlagSet = defaultFlagSet(p.Name)
		}

		// Override the usage text to something nicer.
		p.FlagSet.Usage = func() {
			p.usage(ctx)
		}

		// Parse the flags the user gave us.
		if err := p.FlagSet.Parse(os.Args[1:]); err != nil {
			p.usage(ctx)
			os.Exit(1)
		}

		// Run the main action _if_ we are not in the loop for the version command
		// that is added by default.
		if p.FlagSet.NArg() < 1 || p.FlagSet.Arg(0) != "version" {
			if p.Before != nil {
				if err := p.Before(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
			}

			if err := p.Action(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			// Done.
			return
		}
	}

	// Iterate over the commands in the program.
	for _, command := range p.Commands {
		if name := command.Name(); os.Args[1] == name {
			// Set the default flagset if our flagset is undefined.
			if p.FlagSet == nil {
				p.FlagSet = defaultFlagSet(p.Name)
			}

			// Register the subcommand flags in with the common/global flags.
			command.Register(p.FlagSet)

			// Override the usage text to something nicer.
			p.resetCommandUsage(command)

			// Parse the flags the user gave us.
			if err := p.FlagSet.Parse(os.Args[2:]); err != nil {
				p.FlagSet.Usage()
				os.Exit(1)
			}

			if p.Before != nil {
				if err := p.Before(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
			}

			// Run the command with the context and post-flag-processing args.
			if err := command.Run(ctx, p.FlagSet.Args()); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)

				if p.After != nil {
					if err := p.After(ctx); err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
					}
				}

				os.Exit(1)
			}

			// Run the after function.
			if p.After != nil {
				if err := p.After(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
			}

			// Done.
			return
		}
	}

	fmt.Fprintf(os.Stderr, "%s: no such command\n\n", os.Args[1])
	p.usage(ctx)
	os.Exit(1)
}

func (p *Program) usage(ctx context.Context) error {
	fmt.Fprintf(os.Stderr, "%s -  %s.\n\n", p.Name, strings.TrimSuffix(strings.TrimSpace(p.Description), "."))
	fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", p.Name)
	fmt.Fprintln(os.Stderr)

	// Print information about the common/global flags.
	if p.FlagSet != nil {
		resetFlagUsage(p.FlagSet)
	}

	// Print information about the commands.
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr)

	w := tabwriter.NewWriter(os.Stderr, 0, 4, 2, ' ', 0)
	for _, command := range p.Commands {
		if !command.Hidden() {
			fmt.Fprintf(w, "\t%s\t%s\n", command.Name(), command.ShortHelp())
		}
	}
	w.Flush()

	fmt.Fprintln(os.Stderr)
	return nil
}

func (p *Program) resetCommandUsage(command Command) {
	p.FlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s %s\n", p.Name, command.Name(), command.Args())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, strings.TrimSpace(command.LongHelp()))
		fmt.Fprintln(os.Stderr)
		resetFlagUsage(p.FlagSet)
	}
}

func resetFlagUsage(fs *flag.FlagSet) {
	var (
		hasFlags   bool
		flagBlock  bytes.Buffer
		flagWriter = tabwriter.NewWriter(&flagBlock, 0, 4, 2, ' ', 0)
	)

	fs.VisitAll(func(f *flag.Flag) {
		hasFlags = true

		// Default-empty string vars should read "(default: <none>)"
		// rather than the comparatively ugly "(default: )".
		defValue := f.DefValue
		if defValue == "" {
			defValue = "<none>"
		}

		fmt.Fprintf(flagWriter, "\t-%s\t%s (default: %s)\n", f.Name, f.Usage, defValue)
	})

	flagWriter.Flush()

	if !hasFlags {
		return // Return early.
	}

	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, flagBlock.String())
}

func defaultFlagSet(n string) *flag.FlagSet {
	// Create the default flagset with a debug flag.
	return flag.NewFlagSet(n, flag.ExitOnError)
}
