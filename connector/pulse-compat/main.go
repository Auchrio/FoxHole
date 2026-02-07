package main

import (
	"flag"
	"fmt"
	"os"
	"pulse-compat/utils"
)

func main() {
	cfg := utils.LoadConfig()
	utils.Secret = cfg.Secret

	fs := flag.NewFlagSet("pulse-compat", flag.ContinueOnError)
	listen := fs.Bool("listen", false, "Listen for new message")
	timeout := fs.Int("listen-timeout", -1, "Listen timeout in seconds (0 = indefinite, -1 = config default)")

	// Manual short flag handling
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "-l" {
			args[i] = "-listen"
		} else if arg == "-t" && i+1 < len(args) {
			args[i] = "-listen-timeout"
		}
	}

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pulse-compat <id> [message]\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  pulse-compat myid              # Read latest message\n")
		fmt.Fprintf(os.Stderr, "  pulse-compat -l myid           # Listen for new message (30s default)\n")
		fmt.Fprintf(os.Stderr, "  pulse-compat -l -t 0 myid      # Listen indefinitely\n")
		fmt.Fprintf(os.Stderr, "  pulse-compat myid \"message\"    # Send message\n")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) == 0 {
		fs.Usage()
		os.Exit(1)
	}

	id := parsedArgs[0]

	// Listen mode
	if *listen {
		timeoutSec := cfg.Timeout // use config default
		if *timeout >= 0 {
			timeoutSec = *timeout
		}
		if err := utils.ListenMessages(id, timeoutSec); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Send mode (message provided)
	if len(parsedArgs) > 1 {
		msg := parsedArgs[1]
		if err := utils.SendMessage(id, msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Read mode (just ID)
	if err := utils.ReadMessages(id); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
