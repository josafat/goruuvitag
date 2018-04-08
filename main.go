package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

var influxClient *InfluxDBClient
var deviceCache *cache.Cache

var isPoweredOn = false
var scanMutex = sync.Mutex{}
var Database = "tag_data"

func beginScan(d gatt.Device) {
	scanMutex.Lock()
	for isPoweredOn {
		d.Scan(nil, true) //Scan for five seconds and then restart
		time.Sleep(5 * time.Second)
		d.StopScanning()
	}
	scanMutex.Unlock()
}

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		isPoweredOn = true
		go beginScan(d)
		return
	case gatt.StatePoweredOff:
		log.Println("REINIT ON POWER OFF")
		isPoweredOn = false
		d.Init(onStateChanged)
	default:
		log.Println("WARN: unhandled state: ", string(s))
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if isruuvi, data := ParseRuuviData(a.ManufacturerData, p.ID()); isruuvi {
		fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
		fmt.Println("  TX Power Level    =", a.TxPowerLevel)

		_, found := deviceCache.Get(data.Address)
		if found {
			log.Printf("Not sending until cache cleaned for %s",
				data.Address)
			return
		}

		if err := influxClient.NewMeasurementPoint(data); err != nil {
			log.Printf("WARN: failed to write to influx: %s",
				err.Error())
		}

		deviceCache.Set(data.Address, "dummy", cache.DefaultExpiration)
	}
}

func main() {

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	influxConfig, err := ParseInfluxDBUrl(os.Getenv("INFLUX_URL"), Database)
	if err == nil {
		influxClient, err = NewInfluxDBClient(influxConfig)
		if err != nil {
			log.Fatalf("Failed to setup Influxdb client: %s",
				err.Error())
			return
		}

		log.Printf("Influx client: %#v", influxClient)
	} else {
		log.Printf("Failed to parse url: %s", err.Error())
	}

	deviceCache = cache.New(5*time.Minute, 10*time.Minute)

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered))
	d.Init(onStateChanged)
	select {}
}
