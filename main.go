package main

import (
	"crypto/tls"
	"encoding/json"
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
	// Input flags
	inputHeaders string
	jsonMessage  string
	jsonMethod   string
	textMessage  string

	// Protocol flags
	insecure bool

	// Output flags
	responseOnly bool
	showVersion  bool

	// Verbosity flags
	basic   bool
	verbose bool

	version = "unknown"
)

func init() {
	flag.StringVar(&inputHeaders, "headers", "", "A comma-separated list of headers to send to the target server in the connection establishing request.")
	flag.StringVar(&jsonMessage, "json", "", "A text format JSON RPC message to send to the target server. Response will be printed.")
	flag.StringVar(&jsonMethod, "method", "", "A JSON RPC method to send in a JSON RPC request to the target server. For methods requiring params, use the -json flag. Response will be printed.")
	flag.StringVar(&textMessage, "text", "", "A text message to send to the target server. Response will be printed.")

	flag.BoolVar(&insecure, "insecure", false, "Open an insecure WS connection in the case of no scheme being present in the input URL.")

	// TODO: add flag for binary output, to allow piping to other commands
	flag.BoolVar(&responseOnly, "ro", false, "Response only; print only the response. Has no effect if there's no expected response.")
	flag.BoolVar(&showVersion, "version", false, "Print the version.")

	flag.BoolVar(&basic, "b", false, "Print only basic output.")
	flag.BoolVar(&verbose, "v", false, "Print verbose output, e.g. includes the most important headers.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:  wsstat [options] <url>\n\n")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	if basic && verbose {
		fmt.Print("The basic and verbose flags are mutually exclusive, choose one.\n\n")
		flag.Usage()
		os.Exit(2)
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

	url, err := parseWSURI(args[0])
	if err != nil {
		log.Fatalf("Error parsing input URI: %v", err)
	}

	header := parseHeaders(inputHeaders)
	var result wsstat.Result
	var response interface{}
	if textMessage != "" {
		result, response, err = wsstat.MeasureLatency(url, textMessage, header)
		if err != nil {
			handleConnectionError(err, url.String())
		}
		// TODO: add automatic decoding of detected byte response
	} else if jsonMessage != "" {
		encodedMessage := make(map[string]interface{})
		err := json.Unmarshal([]byte(jsonMessage), &encodedMessage)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON message: %v", err)
		}
		result, response, err = wsstat.MeasureLatencyJSON(url, encodedMessage, header)
		if err != nil {
			handleConnectionError(err, url.String())
		}
	} else if jsonMethod != "" {
		msg := struct {
			Method     string `json:"method"`
			ID         string `json:"id"`
			RPCVersion string `json:"jsonrpc"`
		}{
			Method:     jsonMethod,
			ID:         "1",
			RPCVersion: "2.0",
		}
		result, response, err = wsstat.MeasureLatencyJSON(url, msg, header)
		if err != nil {
			handleConnectionError(err, url.String())
		}
	} else {
		result, err = wsstat.MeasureLatencyPing(url, header)
		if err != nil {
			handleConnectionError(err, url.String())
		}
	}

	// Print the results if there is no expected response or if the responseOnly flag is not set
	if !responseOnly || (jsonMessage == "" && jsonMethod == "" && textMessage == "") {
		// Print details of the request
		printRequestDetails(result)

		// Print the timing results
		printTimingResults(url, result)
	}

	// Print the response, if there is one
	printResponse(response)
}

// colorWSOrange returns the text with a custom orange color.
// The color is from the WS logo, #ff6600 is its hex code.
func colorWSOrange(text string) string {
	return customColor(255, 102, 0, text)
}

// colorTeaGreen returns the text with a custom tea green color.
// The color has hex code #d3f9b5.
func colorTeaGreen(text string) string {
	return customColor(211, 249, 181, text)
}

// customColor returns the text with a custom RGB color.
func customColor(r, g, b int, text string) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, text)
}

// formatPadLeft formats the duration to a string with padding on the left.
func formatPadLeft(d time.Duration) string {
	return fmt.Sprintf("%7dms", int(d/time.Millisecond))
}

// formatPadRight formats the duration to a string with padding on the right.
func formatPadRight(d time.Duration) string {
	return fmt.Sprintf("%-8s", strconv.Itoa(int(d/time.Millisecond))+"ms")
}

// handleConnectionError prints the error message and exits the program.
func handleConnectionError(err error, url string) {
	if strings.Contains(err.Error(), "tls: first record does not look like a TLS handshake") {
		log.Fatalf("Error establishing WS connection to '%s': %v\n\nIs the target server using a secure WS connection? If not, use the '-insecure' flag or specify the correct scheme in the input.", url, err)
	}
	log.Fatalf("Error establishing WS connection to '%s': %v", url, err)
}

// parseHeaders parses the inputHeaders string into an HTTP header.
func parseHeaders(inputHeaders string) http.Header {
	header := http.Header{}
	if inputHeaders != "" {
		headerParts := strings.Split(inputHeaders, ",")
		for _, part := range headerParts {
			parts := strings.Split(part, ":")
			if len(parts) != 2 {
				log.Fatalf("Invalid header format: %s", part)
			}
			header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	return header
}

// parseWSURI parses the rawURI string into a URL object.
func parseWSURI(rawURI string) (*url.URL, error) {
	if !strings.Contains(rawURI, "://") {
		scheme := "wss://"
		if insecure {
			scheme = "ws://"
		}
		rawURI = scheme + rawURI
	}

	url, err := url.Parse(rawURI)
	if err != nil {
		return nil, err
	}

	return url, nil
}

// printRequestDetails prints the headers of the WebSocket connection to the terminal.
func printRequestDetails(result wsstat.Result) {
	fmt.Println()

	// Print basic output
	if basic {
		fmt.Printf("%s: %s\n", colorTeaGreen("URL"), result.URL.Hostname())
		if len(result.IPs) > 0 {
			fmt.Printf("%s:  %s\n", colorTeaGreen("IP"), result.IPs[0])
		}
		return
	}

	// Print verbose output
	if verbose {
		fmt.Println(colorWSOrange("Target"))
		fmt.Printf("  %s:  %s\n", colorTeaGreen("URL"), result.URL.Hostname())
		// Loop in case there are multiple IPs with the target
		for _, ip := range result.IPs {
			fmt.Printf("  %s: %s\n", colorTeaGreen("IP"), ip)
		}
		fmt.Println()
		if result.TLSState != nil {
			fmt.Println(colorWSOrange("TLS"))
			fmt.Printf("  %s: %s\n", colorTeaGreen("Version"), tls.VersionName(result.TLSState.Version))
			fmt.Printf("  %s: %s\n", colorTeaGreen("Cipher Suite"), tls.CipherSuiteName(result.TLSState.CipherSuite))

			// Print the certificate details
			for i, cert := range result.TLSState.PeerCertificates {
				fmt.Printf("  %s: %d\n", colorTeaGreen("Certificate"), i+1)
				fmt.Printf("    Subject: %s\n", cert.Subject)
				fmt.Printf("    Issuer: %s\n", cert.Issuer)
				fmt.Printf("    Not Before: %s\n", cert.NotBefore)
				fmt.Printf("    Not After: %s\n", cert.NotAfter)
			}
			fmt.Println()
		}
		fmt.Println(colorWSOrange("Request headers"))
		for key, values := range result.RequestHeaders {
			fmt.Printf("  %s: %s\n", colorTeaGreen(key), strings.Join(values, ", "))
		}
		fmt.Println(colorWSOrange("Response headers"))
		for key, values := range result.ResponseHeaders {
			fmt.Printf("  %s: %s\n", colorTeaGreen(key), strings.Join(values, ", "))
		}
		return
	}

	// Print standard output
	fmt.Printf("%s: %s\n", colorWSOrange("Target"), result.URL.Hostname())
	for _, values := range result.IPs {
		fmt.Printf("%s: %s\n", colorWSOrange("IP"), values)
	}
	for key, values := range result.RequestHeaders {
		if key == "Sec-WebSocket-Version" {
			fmt.Printf("%s: %s\n", colorWSOrange("WS version"), strings.Join(values, ", "))
		}
	}
	if result.TLSState != nil {
		fmt.Printf("%s: %s\n", colorWSOrange("TLS version"), tls.VersionName(result.TLSState.Version))
	}
}

// printResponse prints the response to the terminal, if there is a response.
func printResponse(response interface{}) {
	if response == nil {
		return
	}
	baseMessage := colorWSOrange("Response") + ": "
	if responseOnly {
		baseMessage = ""
	} else {
		fmt.Println()
	}
	if responseMap, ok := response.(map[string]interface{}); ok {
		// If JSON in request, print response as JSON
		if jsonMessage != "" || jsonMethod != "" {
			responseJSON, err := json.Marshal(responseMap)
			if err != nil {
				fmt.Printf("Could not marshal response to JSON. Response: %v, error: %v", responseMap, err)
				return
			}
			fmt.Printf("%s%s\n", baseMessage, responseJSON)
		} else {
			fmt.Printf("%s%v\n", baseMessage, responseMap)
		}
	} else if responseArray, ok := response.([]interface{}); ok {
		fmt.Printf("%s%v\n", baseMessage, responseArray)
	} else if responseBytes, ok := response.([]byte); ok {
		fmt.Printf("%s%v\n", baseMessage, responseBytes)
	}
	if !responseOnly {
		fmt.Println()
	}
}

// printTimingResults prints the WebSocket statistics to the terminal.
func printTimingResults(url *url.URL, result wsstat.Result) {
	if basic {
		printTimingResultsBasic(result)
	} else {
		printTimingResultsTiered(url, result)
	}
}

// printTimingResultsBasic formats and prints only the most basic WebSocket statistics.
func printTimingResultsBasic(result wsstat.Result) {
	fmt.Println()
	fmt.Printf("%s: %s\n", "Total time", colorWSOrange(strconv.FormatInt(result.TotalTime.Milliseconds(), 10)+"ms"))
	fmt.Println()
}

// printTimingResultsSimple formats and prints the WebSocket statistics to the terminal.
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
func printTimingResultsTiered(url *url.URL, result wsstat.Result) {
	fmt.Println()
	switch url.Scheme {
	case "wss":
		fmt.Fprintf(os.Stdout, wssPrintTemplate,
			colorTeaGreen(formatPadLeft(result.DNSLookup)),
			colorTeaGreen(formatPadLeft(result.TCPConnection)),
			colorTeaGreen(formatPadLeft(result.TLSHandshake)),
			colorTeaGreen(formatPadLeft(result.WSHandshake)),
			colorTeaGreen(formatPadLeft(result.MessageRoundTrip)),
			//formatPadLeft(result.ConnectionClose), // Skipping this for now
			colorTeaGreen(formatPadRight(result.DNSLookupDone)),
			colorTeaGreen(formatPadRight(result.TCPConnected)),
			colorTeaGreen(formatPadRight(result.TLSHandshakeDone)),
			colorTeaGreen(formatPadRight(result.WSHandshakeDone)),
			//formatPadRight(result.FirstMessageResponse), // Skipping due to ConnectionClose skip
			colorWSOrange(formatPadRight(result.TotalTime)),
		)
	case "ws":
		fmt.Fprintf(os.Stdout, wsPrintTemplate,
			colorTeaGreen(formatPadLeft(result.DNSLookup)),
			colorTeaGreen(formatPadLeft(result.TCPConnection)),
			colorTeaGreen(formatPadLeft(result.WSHandshake)),
			colorTeaGreen(formatPadLeft(result.MessageRoundTrip)),
			//formatPadLeft(result.ConnectionClose), // Skipping this for now
			colorTeaGreen(formatPadRight(result.DNSLookupDone)),
			colorTeaGreen(formatPadRight(result.TCPConnected)),
			colorTeaGreen(formatPadRight(result.WSHandshakeDone)),
			//formatPadRight(result.FirstMessageResponse), // Skipping due to ConnectionClose skip
			colorWSOrange(formatPadRight(result.TotalTime)),
		)
	}
	fmt.Println()
}
