package influx

import (
	influxdb1 "github.com/influxdata/influxdb1-client/v2"
	"github.com/max-bytes/metrics-sender/pkg/config"
)

func CreateInfluxConnection(config config.ConfigurationInflux) (influxdb1.Client, error) {
	writeEncoding := influxdb1.DefaultEncoding
	if config.GZip {
		writeEncoding = influxdb1.GzipEncoding
	}
	c, err := influxdb1.NewHTTPClient(influxdb1.HTTPConfig{
		Addr:               config.URL,
		InsecureSkipVerify: true,
		WriteEncoding:      writeEncoding,
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func Send(writePoints []*influxdb1.Point, client influxdb1.Client, config config.ConfigurationInflux) error {

	bp, err := influxdb1.NewBatchPoints(influxdb1.BatchPointsConfig{Database: config.Database})
	if err != nil {
		return err
	}
	bp.AddPoints(writePoints)

	err = client.Write(bp)
	if err != nil {
		return err
	}

	return nil
}
