package main

import (
	"errors"
	"net/url"
	"time"
	"log"

	"github.com/influxdata/influxdb/client/v2"
)

type InfluxDBConfig struct {
	Url      string
	Username string
	Password string
	Database string
	Timeout  time.Duration
}

type InfluxDBClient struct {
	Client client.Client
	Database string
}

func ParseInfluxDBUrl(u string, db string) (*InfluxDBConfig, error) {
	if len(u) < len("http://1.1.1.1") {
		return &InfluxDBConfig{}, errors.New("Invalid influx URL")
	}
	confUrl, err := url.Parse(u)
	if err != nil {
		return &InfluxDBConfig{}, err
	}

	ic := &InfluxDBConfig{
		Url: confUrl.Scheme + "://" + confUrl.Host,
		Database: db,
	}

	ic.Username = confUrl.User.Username()
	ic.Password, _ = confUrl.User.Password()

	return ic, nil
}

func NewInfluxDBClient(config *InfluxDBConfig) (*InfluxDBClient, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.Url,
		Username: config.Username,
		Password: config.Password,
		Timeout:  config.Timeout,
	})
	if err != nil {
		return &InfluxDBClient{}, err
	}

	return &InfluxDBClient{
		Client: c,
		Database: config.Database,
	}, nil

}

func (ic *InfluxDBClient) NewMeasurementPoint(data *SensorData) error {
	if ic == nil {
		return nil
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  ic.Database,
		Precision: "s",
	})
	if err != nil {
		return err
	}

	tags := map[string]string{"address": data.Address}
	fields := map[string]interface{}{
		"temperature":   data.Temp,
		"humidity": data.Humidity,
		"pressure":   data.Pressure,
		"battery":   data.Battery,
	}

	pt, err := client.NewPoint("ruuvisensor", tags, fields, time.Now())
	if err != nil {
		return err
	}
	bp.AddPoint(pt)

	return ic.Client.Write(bp)
}
