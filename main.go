package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jakobilobi/go-wsstat"
)

var (
	// CLI flags
	jsonMessage string
	textMessage string
	showVersion bool

	version = "under development"
)

func init() {
	flag.StringVar(&textMessage, "text", "", "A text message to send to the target server. Response will be printed.")
	flag.StringVar(&jsonMessage, "json", "", "A JSON RPC message to send to the target server. Response will be printed.")
	flag.BoolVar(&showVersion, "v", false, "Print the version.")

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

	if textMessage != "" && jsonMessage != "" {
		fmt.Print("The message options are mutually exclusive, choose one.\n\n")
		flag.Usage()
		os.Exit(2)
	}

	var result wsstat.Result
	var response interface{}
	var err error
	if textMessage != "" {
		result, response, err = wsstat.MeasureLatency(args[0], textMessage)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else if jsonMessage != "" {
		msg := struct {
			Method string `json:"method"`
			Id string `json:"id"`
			RpcVersion string `json:"jsonrpc"`
		}{
			Method: jsonMessage,
			Id: "1",
			RpcVersion: "2.0",
		}
		result, response, err = wsstat.MeasureLatencyJSON(args[0], msg)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		result, err = wsstat.MeasureLatencyPing(args[0])
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// Print the latency
	fmt.Printf("Results: \n%+v", result)

	if responseMap, ok := response.(map[string]interface{}) ; ok {
		fmt.Printf("\nResponse: %v\n", responseMap)
	} else if responseArray, ok := response.([]interface{}) ; ok {
		fmt.Printf("\nResponse: %v\n", responseArray)
	} else if responseBytes, ok := response.([]byte) ; ok {
		fmt.Printf("\nResponse: %v\n", responseBytes)
	}
}
