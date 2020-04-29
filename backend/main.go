package main

import (
	"os"
	"fmt"
	"strings"
	"github.com/cloudflare/cloudflare-go"
	"github.com/joho/godotenv"
	"github.com/gosimple/slug"
	log "github.com/sirupsen/logrus"

	"github.com/LiamBigelow/isfoundat/backend/coords"
	"github.com/LiamBigelow/isfoundat/backend/citadel"
)


func main() {
	// Load ENV vars
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log.SetLevel(log.InfoLevel)

	// Get the list of cities to conform to
	cities := citadel.Parse("../cities.csv")
	for _, c := range cities {
		c.Slug = slug.Make(c.City)
		createDNSData(c)
	}

	// Connect to CloudFlare with token
	api, err := cloudflare.NewWithAPIToken(os.Getenv("CF_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	// Try find the isfound.at zone
	recs, err := api.DNSRecords(os.Getenv("CF_ZONE"), cloudflare.DNSRecord{})
	if err != nil {
	    log.Fatal(err)
	}

	// Create a lookup of existing LOC records
	recMap := make(map[string]*cloudflare.DNSRecord)
	for i, r := range recs {
		if r.Type != "LOC" {
			continue
		}
		baseName := strings.Split(r.Name, ".")[0]
	    recMap[baseName] = &recs[i]
	}

	counter := &Counter{}
	for _, c := range cities {
		processCity(c, recMap[c.Slug], api, counter)
		delete(recMap, c.Slug)
	}

	log.Infof("Cleaning up records")

	cleanOrphanCities(&recMap, api, counter)

	log.WithFields(log.Fields{
		"passed": counter.PassCount,
		"created": counter.NewCount,
		"modified": counter.ModCount,
		"deleted": counter.DelCount,
	}).Infof("Done ✨")
}

func processCity(city *citadel.City, existingRecord *cloudflare.DNSRecord, api *cloudflare.API, counter *Counter) {
	cityLogger := log.WithFields(log.Fields{"slug": city.Slug})

	if existingRecord == nil {
		cityLogger.Debugf("No record found for %s\n", city.City)

		newRecord := getDNSRecord(city)

		if (os.Getenv("RUN_TYPE") == "full") {
			addDNSRecord(api, newRecord)
		} else {
			cityLogger.WithFields(log.Fields{
				"record": newRecord,
			}).Infof("DRY RUN: Would add record for %s\n", newRecord.Name)
		}

		counter.New()
	} else {
		cityLogger.Debugf("Found record for %s\n", city.City)

		needsUpdating := compareLocations(city, city.Data.(map[string]interface{}), existingRecord.Data.(map[string]interface{}))

		if len(needsUpdating) > 0 {
			cityLogger.WithFields(log.Fields{
				"fields": needsUpdating,
			}).Infof("%s needs to be updated\n", city.City)

			existingRecord.Data = city.Data

			if (os.Getenv("RUN_TYPE") == "full") {
				updateDNSRecord(api, existingRecord)
			} else {
				cityLogger.WithFields(log.Fields{
					"record": existingRecord,
				}).Infof("DRY RUN: Would update record to %s\n", existingRecord.Name)
			}

			counter.Modify()
		} else {
			cityLogger.Infof("%s is correct\n", city.City)
			counter.Pass()
		}
	}
}

func compareLocations(city *citadel.City, localFields map[string]interface{}, remoteFields map[string]interface{}) []string {
	incorrectFields := []string{}

	for k, v := range localFields {
		remote := remoteFields[k]
		isEq := v == remote

		if (!isEq) {
			incorrectFields = append(incorrectFields, k)
		}

		log.WithFields(log.Fields{
			"city": city.City,
			"field": k,
			"needsUpdating": !isEq,
			"wewant": v,
			"wefound": remote,
		}).Debug("Testing LOC property")
	}

	return incorrectFields
}

func cleanOrphanCities(cities *map[string]*cloudflare.DNSRecord, api *cloudflare.API, counter *Counter) {
	for citySlug, record := range *cities {
		cityLogger := log.WithFields(log.Fields{"slug": citySlug})
		if (os.Getenv("RUN_TYPE") == "full") {
			deleteDNSRecord(api, record)
		} else {
			cityLogger.Infof("DRY RUN: Would delete record for %s\n", record.Name)
		}
		counter.Delete()
	}
}

// createDNSData formats the city location information into
// the format that matches the CloudFlare API response data
func createDNSData(city *citadel.City) {
	pos, err := coords.ParseDecimalLatLong(city.Latitude, city.Longitude)
    if err != nil {
        log.Fatal(err)
    }

    d := make(map[string]interface{})

    d["altitude"] = float64(city.Altitude)
    d["size"] = float64(city.Diameter)

    d["lat_direction"] = pos.Lat.Dir
    d["lat_degrees"] = float64(pos.Lat.Deg)
    d["lat_minutes"] = float64(pos.Lat.Min)
    d["lat_seconds"] = float64(pos.Lat.Sec)

    d["long_direction"] = pos.Long.Dir
    d["long_degrees"] = float64(pos.Long.Deg)
    d["long_minutes"] = float64(pos.Long.Min)
    d["long_seconds"] = float64(pos.Long.Sec)

    d["precision_horz"] = 0.0
    d["precision_vert"] = 0.0

    city.Data = d
}

func getDNSRecord(city *citadel.City) *cloudflare.DNSRecord {
	return &cloudflare.DNSRecord{
		Type: "LOC",
		Name: fmt.Sprintf("%s.isfound.at", city.Slug),
		TTL: 120,
		Data: city.Data,
	}
}

func addDNSRecord(api *cloudflare.API, record *cloudflare.DNSRecord) {
	resp, err := api.CreateDNSRecord(os.Getenv("CF_ZONE"), *record)
	if err != nil {
	    log.Fatal(err)
	}

	log.WithFields(log.Fields{
		"content": resp.Result.Content,
		"data": resp.Result.Data,
	}).Infof("Added record for %s\n", resp.Result.Name)
}

func updateDNSRecord(api *cloudflare.API, record *cloudflare.DNSRecord) {
	err := api.UpdateDNSRecord(os.Getenv("CF_ZONE"), record.ID, *record)
	if err != nil {
	    log.Fatal(err)
	}

	log.WithFields(log.Fields{
		"content": record.Content,
		"data": record.Data,
	}).Infof("Updated record for %s\n", record.Name)
}

func deleteDNSRecord(api *cloudflare.API, record *cloudflare.DNSRecord) {
	err := api.DeleteDNSRecord(os.Getenv("CF_ZONE"), record.ID)
	if err != nil {
	    log.Fatal(err)
	}

	log.Infof("Deleted record %s\n", record.Name)
}

type Counter struct {
	NewCount int
	ModCount int
	DelCount int
	PassCount int
}

func (c *Counter) New() {
	c.NewCount++
}
func (c *Counter) Modify() {
	c.ModCount++
}
func (c *Counter) Delete() {
	c.DelCount++
}
func (c *Counter) Pass() {
	c.PassCount++
}