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

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"go4.org/netipx"
)

// GeoFeed are properties of a IP geolocation feed - usually in RFC8805 format
type GeoFeed struct {
	key          string // Unique key for the feed, used to generate directory and filenames on disk
	providerName string // Display name of the provider
	url          string // url to slurp it from
	mapbox       struct {
		centerLngLat []float64
		defaultZoom  int
	}
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
		{key: "starlink", providerName: "SpaceX Starlink", url: "https://geoip.starlinkisp.net/feed.csv"},
		{key: "viasat", providerName: "Viasat", url: "https://raw.githubusercontent.com/Viasat/geofeed/main/geofeed.csv"},
	}

	for _, feed := range feeds {
		var locations = make(map[string]netip.Addr) // Country Code ISO2, State or Province ISO2, City

		fmt.Printf("[%s] %s\n", feed.key, feed.url)
		var rows [][]string
		var lmt *time.Time
		var err error
		if rows, lmt, err = readCSVUrl(feed.key, feed.url); err != nil {
			fmt.Printf("[%s] Error reading CSV: %s\n", feed.key, err)
			continue
		}
		// Extract valid subnets from the CSV
		sampleIps := make(map[string][]netip.Addr)
		visibleCountries := make(map[string]bool)
		for _, row := range rows {
			if prefix, err := netip.ParsePrefix(strings.TrimSpace(row[0])); err != nil {
				fmt.Printf("[%s] Error parsing prefix %s: %s\n", feed.key, row[0], err)
			} else {
				locationKey := feedColumnsToKey(row)
				fmt.Printf("[%s] location key = %s\n", feed.key, locationKey)
				// Add a single representative IP address from each subnet to a list of samples.
				// Keep that row's country code/ISO2 too, as it makes the HTML UI more fun.
				if prefix.IsValid() && !prefix.Addr().IsPrivate() && locationKey != "" {
					ip := prefix.Addr()
					// For subnets which aren't single IP address (v4 /32 or v6 /128), we add one IP address to start to get better aesthetics
					if !prefix.IsSingleIP() {
						r := netipx.RangeOfPrefix(prefix)
						ip = r.From().Next()
					}
					cc := strings.ToUpper(strings.TrimSpace(row[1]))
					visibleCountries[cc] = true
					sampleIps[cc] = append(sampleIps[cc], ip)
					locations[locationKey] = ip // this clobbers any older value, but that's fine as we only want one representative IP per country-state-city tuple
				}
			}
		}
		fmt.Printf("[%s] Read %d valid subnets from %d rows of the RFC8805 CSV\n", feed.key, len(sampleIps), len(rows))
		// Prepare directory heirarchy to write JSON files to disk
		dirpath := filepath.Join("..", "gen", "latest-feeds", strings.ToLower(feed.key))
		err = os.MkdirAll(dirpath, 0755)
		if err != nil {
			fmt.Printf("[%s] Error mkdir generated files dir: %s\n", feed.key, err)
			continue
		}

		// Write metadata JSON about the root data source : gen/latest-feeds/<feed-key>/rfc8805.meta.json
		vcl := make([]string, 0, len(visibleCountries))
		for cc := range visibleCountries {
			vcl = append(vcl, cc)
		}
		feedmeta := struct {
			Provider         string   `json:"provider"`
			Url              string   `json:"feedUrl"`
			LastModified     string   `json:"lastModified"`
			VisibleCountries []string `json:"visibleCountries"`
		}{
			feed.providerName,
			feed.url,
			lmt.Format(time.RFC3339),
			vcl,
		}
		var blob []byte
		blob, err = json.MarshalIndent(feedmeta, "", " ")
		if err != nil {
			fmt.Printf("[%s] Error making metadata JSON: %s\n", feed.key, err)
			continue
		}
		err = os.WriteFile(filepath.Join(dirpath, "rfc8805.meta.json"), blob, 0644)
		if err != nil {
			fmt.Printf("[%s] Error writing metadata JSON : %s\n", feed.key, err)
			continue
		}

		// Write JSON file with IP samples and their locations : gen/latest-feeds/<feed-key>/samples.json
		fc := ipToGeoJson(feed.key, feed.providerName, locations)
		for _, f := range fc.Features {
			// this block runs once per country, because each counntry has one and only one feature in the GeoJSON FeatureCollection
			cc := f.Properties["cciso2"].(string)
			if sampleIps[cc] != nil {
				ips := make([]string, 0, len(sampleIps[cc]))
				for _, ip := range sampleIps[cc] {
					ips = append(ips, ip.String())
				}
				f.Properties["ip-samples"] = ips
			}
		}
		gj, _ := json.MarshalIndent(fc, "", " ")
		//fmt.Println(string(gj))
		err = os.WriteFile(filepath.Join(dirpath, "samples.json"), gj, 0644)
		if err != nil {
			fmt.Printf("[%s] Error writing generated JSON IP samples file: %s\n", feed.key, err)
			continue
		}
		// generate map image for a prettier UI
		//for i, cc := range vcl {
		//	vcl[i] = countries.ByName(cc).String()
		//}
		//fmt.Printf("[%s] Generating map image for: %s\n", feed.key, vcl)
		//buildMapImage(vcl, filepath.Join(dirpath, "all-countries.png"), feed.key)

	}
}

// Converts RFC8805 column names to a location key that's suitable for use in a Go map
// First column must be a country code ISO2 Code
// Second column must be a country code ISO2 Code- State or Province ISO2 Code , e.g US-MA or IN-KA
// Third column must be a city name
func feedColumnsToKey(cols []string) string {
	cc := strings.ToUpper(strings.TrimSpace(cols[1]))
	st := strings.ToUpper(strings.TrimSpace(cols[2]))
	city := strings.TrimSpace(cols[3])
	if len(cc) != 2 { // US or IN
		return ""
	}
	if len(st) > 5 { // US-MA or just MA
		return ""
	}
	splitstate := strings.Split(st, "-")
	if len(splitstate) > 1 {
		st = splitstate[1] // for US-MA, we want MA
	} else {
		st = splitstate[0] // for MA, we want MA
	}
	return strings.Join([]string{cc, st, city}, ",")
}

type fastahResponse struct {
	IP              string `json:"ip"`
	IsEuropeanUnion bool   `json:"isEuropeanUnion"`
	L10N            struct {
		CurrencyName   string   `json:"currencyName"`
		CurrencyCode   string   `json:"currencyCode"`
		CurrencySymbol string   `json:"currencySymbol"`
		LangCodes      []string `json:"langCodes"`
	} `json:"l10n"`
	LocationData struct {
		CountryName    string  `json:"countryName"`
		CountryCode    string  `json:"countryCode"`
		CityName       string  `json:"cityName"`
		CityGeonamesID int     `json:"cityGeonamesId"`
		Lat            float64 `json:"lat"`
		Lng            float64 `json:"lng"`
		Tz             string  `json:"tz"`
		ContinentCode  string  `json:"continentCode"`
	} `json:"locationData"`
	Satellite struct {
		Provider string `json:"provider"`
	} `json:"satellite"`
}

// ipToGeoJson makes API calls to the remote Fastah service and maps IP addresses to locations inside a GeoJSON fit for rendering on a map
func ipToGeoJson(key string, providerLabel string, locations map[string]netip.Addr) *geojson.FeatureCollection {
	// Convert the map of sample IP addresses to a map of reverse-geocoded locations
	fastahKey := os.Getenv("FASTAH_PRIVATE_API_KEY") // Not for use with browser-side requests
	fc := geojson.NewFeatureCollection()
	var countries = make(map[string]*geojson.Feature) // key = countryCode , value = a GeoJSON feature with per-country properties and one MultiPoint collection of locations
	var c = &http.Client{Timeout: 5 * time.Second}
	for uniqueloc, ip := range locations {
		fmt.Printf("[%s] Processing loc %s\n", key, uniqueloc)
		var req *http.Request
		var resp *http.Response
		var err error
		// Fastah lookup to provide a lat/long for the IP address
		req, err = http.NewRequest("GET", fmt.Sprintf("https://ep.api.getfastah.com/whereis/v1/json/%s", ip.String()), nil)
		if err != nil {
			fmt.Printf("[%s] Error preparing request for Fastah IP Geolocation API: %v\n", key, err)
			continue
		}
		req.Header.Set("Fastah-Key", fastahKey)
		resp, err = c.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Printf("[%s] Error calling Fastah IP Geolocation API for IP %s: %v\n", key, ip.String(), err)
			panic("API call error")
		}
		defer resp.Body.Close()
		var fr fastahResponse
		err = json.NewDecoder(resp.Body).Decode(&fr)
		if err != nil {
			fmt.Printf("[%s] Error parsing Fastah IP Geolocation API response IP %s: %v\n", key, ip.String(), err)
			continue
		}
		fmt.Printf("[%s] Fastah IP Geolocation API reports RFC8805 entry %s/%s maps to %+v\n", key, ip.String(), uniqueloc, fr)
		if countries[fr.LocationData.CountryCode] == nil {
			f := geojson.NewFeature(orb.MultiPoint{})
			f.Properties["cciso2"] = fr.LocationData.CountryCode
			f.Properties["countryName"] = fr.LocationData.CountryName
			f.Properties["marker-color"] = colorForBrand(key)
			f.Properties["marker-size"] = "large"
			f.Properties["title"] = providerLabel
			f.Properties["description"] = "Approximate location as advertised by " + providerLabel
			countries[fr.LocationData.CountryCode] = f
		}
		var mp orb.MultiPoint = countries[fr.LocationData.CountryCode].Geometry.(orb.MultiPoint)
		mp = append(mp, orb.Point{fr.LocationData.Lng, fr.LocationData.Lat})
		countries[fr.LocationData.CountryCode].Geometry = mp
	}

	// Assemble a FeatureCollection with one Feature per country, and each Feature having a MultiPoint representing all the markers/points within that country
	for _, v := range countries {
		fc.Append(v)
	}
	return fc
}

// colorForBrand helps us decide map marker colors based on brand
func colorForBrand(brand string) string {
	switch brand {
	case "starlink":
		// starlink brand colours are white on black backgrounds, and black on white bg.
		return "#5A5A5A"
	case "viasat":
		// viasat blue
		return "#009FE3"
	default:
		return "#009FE3"
	}
}
