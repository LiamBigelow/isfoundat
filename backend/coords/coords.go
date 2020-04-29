package coords

import (
	"math"
)

type Position struct {
	Lat *DMS
	Long *DMS
}

type DMS struct {
	Deg		uint8
	Min		uint8
	Sec 	float64
	Dir 	string
}

func ParseDecimalLatLong(lat, long float64) (*Position, error) {
	position := Position{}

	position.Lat = timeMaster(lat)
	position.Long = timeMaster(long)

	position.Lat.Dir = getCardinalLatitude(lat)
	position.Long.Dir = getCardinalLongitude(long)

	return &position, nil
}

func timeMaster(decimal float64) *DMS {
	dms := DMS{}

	decimal = math.Abs(decimal)

	dms.Deg = uint8(decimal)
	dms.Min = uint8((decimal - float64(dms.Deg)) * 60)
	dms.Sec = (decimal - float64(dms.Deg) - float64(dms.Min)/60) * 3600
	dms.Sec = math.Round(dms.Sec*1000)/1000

	if (dms.Sec == 60) {
		dms.Sec = 0
		dms.Min++
	}

	return &dms
}

func getCardinalLatitude(decimal float64) string {
	if decimal > 0 {
		return "N"
	}
	return "S"
}

func getCardinalLongitude(decimal float64) string {
	if decimal > 0 {
		return "E"
	}
	return "W"
}