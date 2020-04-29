package citadel

import (
	"os"
	"github.com/gocarina/gocsv"
)

type City struct {
	City      		string 		`csv:"city"`
	Slug			string 		`csv:"-"`
	Latitude    	float64 	`csv:"lat"`
	Longitude   	float64 	`csv:"long"`
	Country 		string 		`csv:"country"`
	Altitude 		float64 	`csv:"alt"`
	Diameter 		float64 	`csv:"diameter"`
	Data			interface{} `csv:"-"`
}

func Parse(filename string) (cities []*City) {
	citiesFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer citiesFile.Close()

	if err := gocsv.UnmarshalFile(citiesFile, &cities); err != nil {
		panic(err)
	}

	return
}