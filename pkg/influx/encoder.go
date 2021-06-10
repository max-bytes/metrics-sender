package influx

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	// protocol "github.com/influxdata/line-protocol"
	influxdb1 "github.com/influxdata/influxdb1-client/v2"
)

func EncodeInfluxLines(variableTags map[string]string) ([]*influxdb1.Point, error) {

	state, err := strconv.Atoi(variableTags["state"])
	if err != nil {
		return nil, fmt.Errorf("Could not parse state %s into integer: %v", variableTags["state"], err)
	}
	delete(variableTags, "state")

	perfdata := variableTags["perfdata"]
	delete(variableTags, "perfdata")

	timestampInt, err := strconv.ParseInt(variableTags["timestamp"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Could not parse timestamp %s into integer: %v", variableTags["timestamp"], err)
	}
	delete(variableTags, "timestamp")
	timestamp := time.Unix(timestampInt, 0)

	metricPoints, err := perfData2Points(perfdata, variableTags, timestamp)
	if err != nil {
		return nil, err
	}

	// add state as its own point, with the state encoded as an integer (0 to 3)
	statePoint, err := state2point("state", state, variableTags, timestamp)
	if err != nil {
		return nil, err
	}
	allPoints := append(metricPoints, statePoint)

	return allPoints, nil
}

func state2point(metricName string, state int, addedTags map[string]string, timestamp time.Time) (*influxdb1.Point, error) {
	var fields = map[string]interface{}{
		"value": state,
	}

	point, err := influxdb1.NewPoint(metricName, addedTags, fields, timestamp)
	if err != nil {
		return nil, err
	}

	return point, nil
}

// partly taken from https://github.com/Griesbacher/nagflux/blob/ea877539bc49ed67e9a5e35b8a127b1ff4cadaad/collector/spoolfile/nagiosSpoolfileWorker.go
var regexPerformanceLabel = regexp.MustCompile(`([^=]+)=(U|[\d\.,\-]+)([\pL\/%]*);?([\d\.,\-:~@]+)?;?([\d\.,\-:~@]+)?;?([\d\.,\-]+)?;?([\d\.,\-]+)?;?\s*`)

func perfData2Points(str string, addedTags map[string]string, timestamp time.Time) ([]*influxdb1.Point, error) {
	perfSlices := regexPerformanceLabel.FindAllStringSubmatch(str, -1)

	points := make([]*influxdb1.Point, 0, len(perfSlices))
	for _, perfSlice := range perfSlices {
		label := perfSlice[1]

		v, err := strconv.ParseFloat(perfSlice[2], 64)
		if err != nil {
			fmt.Println(err)
			continue
		}

		var fields = map[string]interface{}{
			"value": v,
		}
		var tags = map[string]string{
			"label": label,
		}
		for tagKey, tagValue := range addedTags {
			tags[tagKey] = tagValue
		}

		// add UOM to tags, if present
		if perfSlice[3] != "" {
			tags["uom"] = perfSlice[3]
		}
		warnF, err := strconv.ParseFloat(perfSlice[4], 64)
		if err == nil {
			fields["warn"] = warnF
		}
		critF, err := strconv.ParseFloat(perfSlice[5], 64)
		if err == nil {
			fields["crit"] = critF
		}
		minF, err := strconv.ParseFloat(perfSlice[6], 64)
		if err == nil {
			fields["min"] = minF
		}
		maxF, err := strconv.ParseFloat(perfSlice[7], 64)
		if err == nil {
			fields["max"] = maxF
		}

		point, err := influxdb1.NewPoint("metric", tags, fields, timestamp)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}
