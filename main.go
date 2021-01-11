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

var akamaiIPRanges = "104.101.221.0/24 184.51.125.0/24 184.51.154.0/24 184.51.157.0/24 184.51.33.0/24 2.16.36.0/24 2.16.37.0/24 2.22.226.0/24 2.22.227.0/24 2.22.60.0/24 23.15.12.0/24 23.15.13.0/24 23.209.105.0/24 23.62.225.0/24 23.74.29.0/24 23.79.224.0/24 23.79.225.0/24 23.79.226.0/24 23.79.227.0/24 23.79.229.0/24 23.79.230.0/24 23.79.231.0/24 23.79.232.0/24 23.79.233.0/24 23.79.235.0/24 23.79.237.0/24 23.79.238.0/24 23.79.239.0/24 63.208.195.0/24 72.246.0.0/24 72.246.1.0/24 72.246.116.0/24 72.246.199.0/24 72.246.2.0/24 72.247.150.0/24 72.247.151.0/24 72.247.216.0/24 72.247.44.0/24 72.247.45.0/24 80.67.64.0/24 80.67.65.0/24 80.67.70.0/24 80.67.73.0/24 88.221.208.0/24 88.221.209.0/24 96.6.114.0/24"
var cloudflareIPRanges = "173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/12 172.64.0.0/13 131.0.72.0/22"
var incapsulaIPRanges = "199.83.128.0/21 198.143.32.0/19 149.126.72.0/21 103.28.248.0/22 45.64.64.0/22 185.11.124.0/22 192.230.64.0/18 107.154.0.0/16 45.60.0.0/16 45.223.0.0/16"
var sucuriIPRanges = "185.93.228.0/24 185.93.229.0/24 185.93.230.0/24 185.93.231.0/24 192.124.249.0/24 192.161.0.0/24 192.88.134.0/24 192.88.135.0/24 193.19.224.0/24 193.19.225.0/24 66.248.200.0/24 66.248.201.0/24 66.248.202.0/24 66.248.203.0/24"

var outputToSave = []string{}
var out io.Writer = os.Stdout
var quietMode bool
var ipMode bool

func main() {
	var outputFileFlag string
	flag.StringVar(&outputFileFlag, "o", "", "Output a list of the identified IP addresses with their URL and the provider (if identified)")
	quietModeFlag := flag.Bool("q", false, "Only output the data we care about")
	ipModeFlag := flag.Bool("i", false, "Input is already a list of IP addresses")
	flag.Parse()

	quietMode = *quietModeFlag
	ipMode = *ipModeFlag
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

	akamaiRanges := strings.Split(akamaiIPRanges, " ")
	cloudflareRanges := strings.Split(cloudflareIPRanges, " ")
	incapsulaRanges := strings.Split(incapsulaIPRanges, " ")
	sucuriIPRanges := strings.Split(sucuriIPRanges, " ")

	for u := range targetDomains {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if !quietMode {
				fmt.Println("Checking:", url)
			}

			identifiedIPs := []string{}
			if ipMode {
				identifiedIPs = append(identifiedIPs, url)
			} else {
				identifiedIPs = getIPForDomain(url)
			}

			if len(identifiedIPs) > 0 {
				for _, i := range identifiedIPs {
					wasFoundInCloud := false
					wasFoundInCloud = checkIPInRange(akamaiRanges, i, url, "akamai")

					if !wasFoundInCloud {
						wasFoundInCloud = checkIPInRange(cloudflareRanges, i, url, "cloudflare")
					}

					if !wasFoundInCloud {
						wasFoundInCloud = checkIPInRange(incapsulaRanges, i, url, "incapsula")
					}

					if !wasFoundInCloud {
						wasFoundInCloud = checkIPInRange(sucuriIPRanges, i, url, "sucuri")
					}

					if !wasFoundInCloud {
						newLine := url + "|" + i + "|n/a"
						fmt.Println(newLine)
						outputToSave = append(outputToSave, newLine)
					}
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
	fmt.Println("Vape -> Crawl3r")
	fmt.Println("Checks to see if a URL is hosted behind a cloud provider. Reads directly from stdin.")
	fmt.Println("")
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

func checkIPInRange(rangeCollection []string, target string, url string, rangeProvider string) bool {
	targetIP := net.ParseIP(target)
	if !quietMode {
		fmt.Println("Checking IP:", targetIP, "in range for", rangeProvider)
	}

	wasFoundInCloud := false
	for _, r := range rangeCollection {
		_, rangeIPNet, _ := net.ParseCIDR(r)
		// is the target IP in the first range?
		if rangeIPNet.Contains(targetIP) {
			if !quietMode {
				fmt.Printf("[+] %s exists in %s\n", target, rangeProvider)
			}

			newLine := url + "|" + target + "|" + rangeProvider
			fmt.Println(newLine)
			outputToSave = append(outputToSave, newLine)

			wasFoundInCloud = true
			break
		}
	}

	return wasFoundInCloud
}

func readStdin() <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			url := strings.ToLower(sc.Text())
			if url != "" {
				// strip the http:// or https:// here other the IP look up fails
				// Note: we don't care for multiple entries of the same URL
				final := strings.Replace(url, "http://", "", -1)
				final = strings.Replace(final, "https://", "", -1)
				lines <- final
			}
		}
	}()
	return lines
}
