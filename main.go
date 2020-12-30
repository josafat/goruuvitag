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

var deviceCache *cache.Cache

var isPoweredOn = false
var scanMutex = sync.Mutex{}
var Database = "tag_data"
var debug = false

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
		if debug {
			fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
			fmt.Println("  TX Power Level    =", a.TxPowerLevel)
		}

		deviceCache.Set(data.Address, data, cache.DefaultExpiration)
	}
}

func main() {

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	deviceCache = cache.New(5*time.Minute, 10*time.Minute)

	restPort := os.Getenv("REST_PORT")
	if len(restPort) > 0 {
		restServer, err := NewRestServer(restPort, deviceCache)
		if err != nil {
			log.Fatalf("rest server failed: %s",
				err.Error())
			return
		}

		go restServer.Run()
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered))
	d.Init(onStateChanged)
	select {}
}
