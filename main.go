package main

import (
	"flag"
	"fmt"

	"github.com/jakobilobi/go-wsstat"
)

func main() {
	flag.Parse()

	// Get the first argument
	args := flag.Args()
	if len(args) == 1 {
		// Print the first argument
		fmt.Printf("Input arg: %s\n\n", args[0])
	} else {
		fmt.Println("No or too many input arg")
		return
	}

	result, _, err := wsstat.MeasureLatency(args[0], "testing")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Print the latency
	fmt.Printf("Results: \n%+v", result)
}
