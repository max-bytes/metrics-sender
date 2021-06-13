package parser

import (
	"fmt"
	"regexp"

	"mhx.at/gitlab/landscape/metrics-sender/pkg/influx"

	influxdb1 "github.com/influxdata/influxdb1-client/v2"
)

var tokenRegex = regexp.MustCompile(`(?m)(.*?)::(.*)`)
var delimiterRegex = regexp.MustCompile(`\!\*\*\!\*\!\*\*\!`)

func Parse(lines []string) ([]*influxdb1.Point, error) {

	var pointsInFile = []*influxdb1.Point{}

	// process file line-by-line
	for _, line := range lines {
		tokens := delimiterRegex.Split(line, -1)
		fields := make(map[string]string)
		for _, token := range tokens {
			subTokens := tokenRegex.FindStringSubmatch(token)
			if len(subTokens) != 3 {
				return nil, fmt.Errorf("Could not parse token %s: invalid number of subtokens", token)
			}
			fields[subTokens[1]] = subTokens[2]
		}

		pointsOfLine, err := influx.EncodeInfluxLines(fields)
		if err != nil {
			return nil, fmt.Errorf("Could not encode influx line %s: %v", line, err)
		}
		pointsInFile = append(pointsInFile, pointsOfLine...)
	}
	return pointsInFile, nil
}
