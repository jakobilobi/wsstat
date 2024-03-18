package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jakobilobi/go-wsstat"
)

var (
	// CLI flags
	showVersion bool
	message     string

	version = "under development"
)

func init() {
	flag.StringVar(&message, "m", "", "[optional] The message to send to the target server")
	flag.BoolVar(&showVersion, "v", false, "Print the program's version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <url>\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		fmt.Print("Use only a single input argument, the URL of the target server.\n\n")
		flag.Usage()
		os.Exit(2)
	}

	result, _, err := wsstat.MeasureLatency(args[0], "testing")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the latency
	fmt.Printf("Results: \n%+v", result)
}
