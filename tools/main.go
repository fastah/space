package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go4.org/netipx"
)

// GeoFeed are properties of a IP geolocation feed - usually in RFC8805 format
type GeoFeed struct {
	Key          string // Unique key for the feed, used to generate directory and filenames on disk
	ProviderName string // Display name of the provider
	Url          string // url to slurp it from
}

// IP is a sample IP address from the feed file
type IP struct {
	Ip netip.Addr `json:"ip"`
	CC string     `json:"cciso2"`
}

func readCSVUrl(key, url string) ([][]string, *time.Time, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	lmt, err := http.ParseTime(resp.Header.Get("last-modified"))
	if err != nil {
		fmt.Printf("[%s] No last-modified header sent by server, using current time\n", key)
		lmt = time.Now()
	}
	reader := csv.NewReader(resp.Body)
	reader.Comma = ','
	reader.Comment = '#'
	data, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	return data, &lmt, nil
}

func main() {
	feeds := []GeoFeed{
		{Key: "starlink", ProviderName: "Starlink by SpaceX", Url: "https://geoip.starlinkisp.net/feed.csv"},
		{Key: "viasat", ProviderName: "Viasat", Url: "https://raw.githubusercontent.com/Viasat/geofeed/main/geofeed.csv"},
	}

	for _, feed := range feeds {
		fmt.Printf("[%s] %s\n", feed.Key, feed.Url)
		var rows [][]string
		var lmt *time.Time
		var err error
		if rows, lmt, err = readCSVUrl(feed.Key, feed.Url); err != nil {
			fmt.Printf("[%s] Error reading CSV: %s\n", feed.Key, err)
			continue
		}
		// Extract valid subnets from the CSV
		sampleIps := make(map[string][]netip.Addr)
		visibleCountries := make(map[string]bool)
		for _, row := range rows {
			if prefix, err := netip.ParsePrefix(strings.TrimSpace(row[0])); err != nil {
				fmt.Printf("[%s] Error parsing prefix %s: %s\n", feed.Key, row[0], err)
			} else {
				// Add a single representative IP address from each subnet to a list of samples.
				// Keep that row's country code/ISO2 too, as it makes the HTML UI more fun.
				if prefix.IsValid() && !prefix.Addr().IsPrivate() && len(strings.TrimSpace(row[1])) == 2 {
					ip := prefix.Addr()
					// For subnets which aren't single IP address (v4 /32 or v6 /128), we add one IP address to start to get better aesthetics
					if !prefix.IsSingleIP() {
						r := netipx.RangeOfPrefix(prefix)
						ip = r.From().Next()
					}
					cc := strings.ToUpper(strings.TrimSpace(row[1]))
					visibleCountries[cc] = true
					sampleIps[cc] = append(sampleIps[cc], ip)
				}
			}
		}
		fmt.Printf("[%s] Read %d valid subnets from %d rows of the RFC8805 CSV\n", feed.Key, len(sampleIps), len(rows))
		// Prepare directory heirarchy to write JSON files to disk
		dirpath := filepath.Join("..", "gen", "latest-feeds", strings.ToLower(feed.Key))
		err = os.MkdirAll(dirpath, 0755)
		if err != nil {
			fmt.Printf("[%s] Error mkdir generated files dir: %s\n", feed.Key, err)
			continue
		}
		var blob []byte
		// Write sample IPs list as JSON to disk: gen/latest-feeds/<feed-key>/samples.json
		blob, err = json.MarshalIndent(sampleIps, "", "  ")
		if err != nil {
			fmt.Printf("[%s] Error marshalling JSON: %s\n", feed.Key, err)
			continue
		}
		err = os.WriteFile(filepath.Join(dirpath, "samples.json"), blob, 0644)
		if err != nil {
			fmt.Printf("[%s] Error writing generated JSON IP samples file: %s\n", feed.Key, err)
			continue
		}
		// Write metadata JSON about the root data source : gen/latest-feeds/<feed-key>/rfc8805.meta.json
		visibleCountriesList := make([]string, 0, len(visibleCountries))
		for cc := range visibleCountries {
			visibleCountriesList = append(visibleCountriesList, cc)
		}
		feedmeta := struct {
			Provider         string   `json:"provider"`
			Url              string   `json:"feedUrl"`
			LastModified     string   `json:"lastModified"`
			VisibleCountries []string `json:"visibleCountries"`
		}{
			feed.ProviderName,
			feed.Url,
			lmt.Format(time.RFC3339),
			visibleCountriesList,
		}
		blob, err = json.MarshalIndent(feedmeta, "", " ")
		if err != nil {
			fmt.Printf("[%s] Error making metadata JSON: %s\n", feed.Key, err)
			continue
		}
		err = os.WriteFile(filepath.Join(dirpath, "rfc8805.meta.json"), blob, 0644)
		if err != nil {
			fmt.Printf("[%s] Error writing metadata JSON : %s\n", feed.Key, err)
			continue
		}

	}
}
