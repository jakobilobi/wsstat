package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jakobilobi/go-wsstat"
)

const (
	wssPrintTemplate = `` +
		`  DNS Lookup    TCP Connection    TLS Handshake    WS Handshake    Message RTT` + "\n" +
		`|%s  |      %s  |     %s  |    %s  |   %s  |` + "\n" +
		`|           |                 |                |               |              |` + "\n" +
		`|  DNS lookup:%s        |                |               |              |` + "\n" +
		`|                 TCP connected:%s       |               |              |` + "\n" +
		`|                                       TLS done:%s      |              |` + "\n" +
		`|                                                        WS done:%s     |` + "\n" +
		`-                                                                         Total:%s` + "\n"

	wsPrintTemplate = `` +
	`  DNS Lookup    TCP Connection    WS Handshake    Message RTT` + "\n" +
	`|%s  |      %s  |    %s  |  %s   |` + "\n" +
	`|           |                 |               |              |` + "\n" +
	`|  DNS lookup:%s        |               |              |` + "\n" +
	`|                 TCP connected:%s      |              |` + "\n" +
	`|                                       WS done:%s     |` + "\n" +
	`-                                                        Total:%s` + "\n"
)

var (
	// CLI flags
	jsonMessage  string
	textMessage  string
	insecure     bool
	showVersion  bool
	verbose      bool
	extraVerbose bool

	version = "under development"

	verbosity = 0
)

func init() {
	flag.StringVar(&textMessage, "text", "", "A text message to send to the target server. Response will be printed.")
	flag.StringVar(&jsonMessage, "json", "", "A JSON RPC message to send to the target server. Response will be printed.")
	flag.BoolVar(&insecure, "insecure", false, "Open an insecure WS connection in the case of no scheme being present in the input.")
	flag.BoolVar(&showVersion, "version", false, "Print the version.")
	flag.BoolVar(&verbose, "v", false, "Print verbose output, e.g. includes the most important headers.")
	flag.BoolVar(&extraVerbose, "vv", false, "Print extra verbose output, e.g. includes all headers and certificates.")

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

	// Parse the verbosity level
	if extraVerbose {
		verbosity = 2
	} else if verbose {
		verbosity = 1
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

	// Print details of the request
	printRequestDetails(result)

	// Print the timing results
	printTimingResultsTiered(url, result)

	// Print the response, if there is one
	printResponse(response)
}

// formatPadLeft formats the duration to a string with padding on the left.
func formatPadLeft(d time.Duration) string {
	return fmt.Sprintf("%7dms", int(d/time.Millisecond))
}

// formatPadRight formats the duration to a string with padding on the right.
func formatPadRight(d time.Duration) string {
	return fmt.Sprintf("%-8s", strconv.Itoa(int(d/time.Millisecond))+"ms")
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

// printRequestDetails prints the headers of the WebSocket connection to the terminal.
// TODO: consider adding some color to make the output more readable
// TODO: add remote address when available from Result
// TODO: add certificate details
func printRequestDetails(result wsstat.Result) {
	if verbosity == 0 {
		// TODO: print only the most basic info here, maybe target IP, port and nothing else?
		return
	}
	fmt.Println()
	if verbosity == 1 {
		fmt.Println("Request information")
		if result.TLSState != nil {
			fmt.Printf("  TLS version: %s\n", tls.VersionName(result.TLSState.Version))
		}
		for key, values := range result.RequestHeaders {
			if key == "Origin" {
				fmt.Printf("  Origin: %s\n", strings.Join(values, ", "))
			}
			if key == "Sec-WebSocket-Version" {
				fmt.Printf("  WebSocket version: %s\n", strings.Join(values, ", "))
			}
		}
		return
	}
	if verbosity == 2 {
		if result.TLSState != nil {
			fmt.Println("TLS")
			fmt.Printf("  Version: %s\n", tls.VersionName(result.TLSState.Version))
			fmt.Printf("  Cipher Suite: %s\n", tls.CipherSuiteName(result.TLSState.CipherSuite))

			// Print the certificate details
			for i, cert := range result.TLSState.PeerCertificates {
				fmt.Printf("  Certificate %d\n", i+1)
				fmt.Printf("    Subject: %s\n", cert.Subject)
				fmt.Printf("    Issuer: %s\n", cert.Issuer)
				fmt.Printf("    Not Before: %s\n", cert.NotBefore)
				fmt.Printf("    Not After: %s\n", cert.NotAfter)
			}
			fmt.Println()
		}
		fmt.Println("Request headers")
		for key, values := range result.RequestHeaders {
			fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
		}
		fmt.Println("Response headers")
		for key, values := range result.ResponseHeaders {
			fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
		}
		return
	}
}

// printResponse prints the response to the terminal, if there is a response.
// TODO: consider adding some color to make the output more readable
func printResponse(response interface{}) {
	if response == nil {
		return
	}
	fmt.Println()
	if responseMap, ok := response.(map[string]interface{}) ; ok {
		// TODO: print the response in a more readable format
		fmt.Printf("Response: %v\n", responseMap)
	} else if responseArray, ok := response.([]interface{}) ; ok {
		fmt.Printf("Response: %v\n", responseArray)
	} else if responseBytes, ok := response.([]byte) ; ok {
		fmt.Printf("Response: %v\n", responseBytes)
	}
	fmt.Println()
}

// printTimingResultsSimple formats and prints the WebSocket statistics to the terminal.
// TODO: consider adding some color to make the output more readable
func printTimingResultsSimple(result wsstat.Result) {
	const padding = 2
	fmt.Println()

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

// printTimingResultsTiered formats and prints the WebSocket statistics to the terminal in a tiered fashion.
// TODO: consider adding some color to make the output more readable
func printTimingResultsTiered(url *url.URL ,result wsstat.Result) {
	fmt.Println()
	switch url.Scheme {
	case "wss":
		fmt.Fprintf(os.Stdout, wssPrintTemplate,
			formatPadLeft(result.DNSLookup),
			formatPadLeft(result.TCPConnection),
			formatPadLeft(result.TLSHandshake),
			formatPadLeft(result.WSHandshake),
			formatPadLeft(result.MessageRoundTrip),
			//formatPadLeft(result.ConnectionClose), // Skipping this for now
			formatPadRight(result.DNSLookupDone),
			formatPadRight(result.TCPConnected),
			formatPadRight(result.TLSHandshakeDone),
			formatPadRight(result.WSHandshakeDone),
			//formatPadRight(result.FirstMessageResponse), // Skipping due to ConnectionClose skip
			formatPadRight(result.TotalTime),
		)
	case "ws":
		fmt.Fprintf(os.Stdout, wsPrintTemplate,
			formatPadLeft(result.DNSLookup),
			formatPadLeft(result.TCPConnection),
			formatPadLeft(result.WSHandshake),
			formatPadLeft(result.MessageRoundTrip),
			//formatPadLeft(result.ConnectionClose), // Skipping this for now
			formatPadRight(result.DNSLookupDone),
			formatPadRight(result.TCPConnected),
			formatPadRight(result.WSHandshakeDone),
			//formatPadRight(result.FirstMessageResponse), // Skipping due to ConnectionClose skip
			formatPadRight(result.TotalTime),
		)
	}
}