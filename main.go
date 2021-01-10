package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

/*
	Input: cat a list of URLs/IPs via stdin

	Aim: Figure out the IP address of the target and check it against IP ranges to see if we are on a cloud provider or not

	Reason: I normally do this in bash by chaining various utilities. But I hit I/O problems and Unix is hard. Go is not.
*/

var cloudflareIPRanges = "173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/12 172.64.0.0/13 131.0.72.0/22"

var outputToSave = []string{}
var out io.Writer = os.Stdout

func main() {
	var outputFileFlag string
	flag.StringVar(&outputFileFlag, "o", "", "Output a list of the identified IP addresses with their URL and the provider (if identified)")
	quietModeFlag := flag.Bool("q", false, "Only output the data we care about")
	flag.Parse()

	quietMode := *quietModeFlag
	saveOutput := outputFileFlag != ""

	if !quietMode {
		banner()
		fmt.Println("")
	}

	writer := bufio.NewWriter(out)
	targetDomains := make(chan string, 1)
	var wg sync.WaitGroup

	ch := readStdin()
	go func() {
		//translate stdin channel to domains channel
		for u := range ch {
			targetDomains <- u
		}
		close(targetDomains)
	}()

	// flush to writer periodically
	t := time.NewTicker(time.Millisecond * 500)
	defer t.Stop()
	go func() {
		for {
			select {
			case <-t.C:
				writer.Flush()
			}
		}
	}()

	cloudflareRanges := strings.Split(cloudflareIPRanges, " ")

	for u := range targetDomains {
		fmt.Println("Test:", u)
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			fmt.Println("Checking:", url)
			identifiedIPs := getIPForDomain(url)
			if len(identifiedIPs) > 0 {
				for _, i := range identifiedIPs {
					checkIPInRange(cloudflareRanges, i, url, "cloudflare")
				}
			}
		}(u)
	}

	wg.Wait()

	// just in case anything is still in buffer
	writer.Flush()

	if saveOutput {
		file, err := os.OpenFile(outputFileFlag, os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil && !quietMode {
			log.Fatalf("failed creating file: %s", err)
		}

		datawriter := bufio.NewWriter(file)

		for _, data := range outputToSave {
			_, _ = datawriter.WriteString(data + "\n")
		}

		datawriter.Flush()
		file.Close()
	}
}

func banner() {
	fmt.Println("---------------------------------------------------")
	fmt.Println("LeakyTap -> Crawl3r")
	fmt.Println("List URL's which appear to be leaking source instead of having the server interpret it")
	fmt.Printf("Currently looks for:\n\tphp\n\n")
	fmt.Println("Run again with -q for cleaner output")
	fmt.Println("---------------------------------------------------")
}

func getIPForDomain(url string) []string {
	identifiedIPAddresses := []string{}
	ips, _ := net.LookupIP(url)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			identifiedIPAddresses = append(identifiedIPAddresses, ip.String())
		}
	}
	return identifiedIPAddresses
}

func checkIPInRange(rangeCollection []string, target string, url string, rangeProvider string) {
	targetIP := net.ParseIP(target)

	for _, r := range rangeCollection {
		_, rangeIPNet, _ := net.ParseCIDR(r)
		// is the target IP in the first range?
		if rangeIPNet.Contains(targetIP) {
			fmt.Print("Exists in range!")
			newLine := url + "|" + target + "|" + rangeProvider
			outputToSave = append(outputToSave, newLine)
			break
		}
	}
}

func readStdin() <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			url := strings.ToLower(sc.Text())
			if url != "" {
				lines <- url
			}
		}
	}()
	return lines
}
