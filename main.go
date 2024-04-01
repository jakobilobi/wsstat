package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jakobilobi/go-wsstat"
)

const (
	wssPrintTemplate = `` +
		`  DNS Lookup   TCP Connection   TLS Handshake  WS Handshake   Message Round-Trip   Connection Close` + "\n" +
		`(%s  |     %s  |    %s  |      %s  |      %s  |       %s  )` + "\n" +
		`            |                |         |         |                   |                  |` + "\n" +
		`   DNS lookup:%s      |            |         |                   |                  |` + "\n" +
		`                       TCP connected:%s   |         |              |                  |` + "\n" +
		`                                   TLS done:%s         |        |            |` + "\n" +
		`                                   WS done:%s         |                  |` + "\n" +
		`                                                     Msg returned:%s        |` + "\n" +
		`                                                                                Total:%s` + "\n"

	wsPrintTemplate = `` +
	`  DNS Lookup   TCP Connection  WS Handshake   Message Round-Trip   Connection Close` + "\n" +
	`[%s  |     %s  |    %s  |        %s  |       %s  ]` + "\n" +
	`            |                |               |                   |                  |` + "\n" +
	`   DNS lookup:%s      |               |                   |                  |` + "\n" +
	`                       TCP connected:%s   |                   |                  |` + "\n" +
	`                                   WS done:%s         |                  |` + "\n" +
	`                                                     Msg returned:%s        |` + "\n" +
	`                                                                                Total:%s` + "\n"
)

var (
	// CLI flags
	jsonMessage string
	textMessage string
	insecure	bool
	showVersion bool

	version = "under development"
)

func init() {
	flag.StringVar(&textMessage, "text", "", "A text message to send to the target server. Response will be printed.")
	flag.StringVar(&jsonMessage, "json", "", "A JSON RPC message to send to the target server. Response will be printed.")
	flag.BoolVar(&insecure, "insecure", false, "Open an insecure WS connection in the case of no scheme being present in the input.")
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
		flag.Usage()
		os.Exit(2)
	}

	if textMessage != "" && jsonMessage != "" {
		fmt.Print("The message options are mutually exclusive, choose one.\n\n")
		flag.Usage()
		os.Exit(2)
	}

	url, err := parseWsUri(args[0])
	if err != nil {
		log.Fatalf("Error parsing input URI: %v", err)
		return
	}

	var result wsstat.Result
	var response interface{}
	if textMessage != "" {
		result, response, err = wsstat.MeasureLatency(url, textMessage, http.Header{}) // TODO: make headers configurable
		if err != nil {
			log.Fatalf("Error measuring latency: %v", err)
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
		result, response, err = wsstat.MeasureLatencyJSON(url, msg, http.Header{}) // TODO: make headers configurable
		if err != nil {
			log.Fatalf("Error measuring latency: %v", err)
			return
		}
	} else {
		result, err = wsstat.MeasureLatencyPing(url, http.Header{}) // TODO: make headers configurable
		if err != nil {
			log.Fatalf("Error measuring latency: %v", err)
			return
		}
	}

	printResults(result, response)
}

// parseWsUri parses the rawURI string into a URL object.
func parseWsUri(rawURI string) (*url.URL, error) {
	if !strings.Contains(rawURI, "://") {
		scheme := "wss://"
		if insecure {
			scheme = "ws://"
		}
		rawURI = scheme + rawURI
	}

	url, err := url.Parse(rawURI)
	if err != nil{
		return nil, err
	}

	return url, nil
}

// printResults formats and prints the WebSocket statistics to the terminal.
// TODO: consider adding some color to make the output more readable
func printResults(result wsstat.Result, response interface{}) {
	const padding = 2
	fmt.Println()

	// Header
	// TODO: activate these when the info is available from Result
	/* fmt.Print("Connected to <WS URL>\n\n")
	fmt.Printf("Connected via %s\t\n\n", "<TLS version>") */

	if responseMap, ok := response.(map[string]interface{}) ; ok {
		// TODO: print the response in a more readable format
		fmt.Printf("Response: %v\n\n", responseMap)
	} else if responseArray, ok := response.([]interface{}) ; ok {
		fmt.Printf("Response: %v\n\n", responseArray)
	} else if responseBytes, ok := response.([]byte) ; ok {
		fmt.Printf("Response: %v\n\n", responseBytes)
	}

	// Tab writer to help with formatting a tab-separated output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.TabIndent)

	// Add headers for the printout
	headers := []string{"DNS Lookup", "TCP Connection", "TLS Handshake", "WS Handshake", "Message Round-Trip", "Connection Close"}
	fmt.Fprintln(w, strings.Join(headers, "\t|\t")+"\t")

	// Add the result numbers
	stats := []string{
		fmt.Sprintf("%dms", result.DNSLookup.Milliseconds()),
		fmt.Sprintf("%dms", result.TCPConnection.Milliseconds()),
		fmt.Sprintf("%dms", result.TLSHandshake.Milliseconds()),
		fmt.Sprintf("%dms", result.WSHandshake.Milliseconds()),
		fmt.Sprintf("%dms", result.MessageRoundTrip.Milliseconds()),
		fmt.Sprintf("%dms", result.ConnectionClose.Milliseconds()),
	}
	fmt.Fprintln(w, strings.Join(stats, "\t|\t")+"\t")

	// Write the tabbed output to the writer, flush it to stdout
	if err := w.Flush(); err != nil {
		panic(err)
	}

	// Finally, print the total time
	fmt.Printf("\nTotal time:\t%s\t\n", fmt.Sprintf("%dms", result.TotalTime.Milliseconds()))
}
