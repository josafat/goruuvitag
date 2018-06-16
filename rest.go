package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/patrickmn/go-cache"
)

type RestServer struct {
	Port string
	Cache *cache.Cache
}

func NewRestServer(port string, deviceCache *cache.Cache) (*RestServer, error) {
	_, err := strconv.Atoi(port)
	if err != nil {
		return &RestServer{}, err
	}

	return &RestServer{Port:port, Cache: deviceCache}, nil
}

func (rs *RestServer) Run() {
	log.Printf("starting rest at port %s", rs.Port)

	http.HandleFunc("/device", rs.SensorReadHandler)
	http.HandleFunc("/", rs.DevicesHandler)
	log.Fatal(http.ListenAndServe(":" + rs.Port, nil))
}

func (rs *RestServer) DevicesHandler(w http.ResponseWriter, r *http.Request) {
	items := rs.Cache.Items()

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(true)
	if err := enc.Encode(items); err != nil {
		log.Printf("Failed to encode results: %s", err.Error())
		http.Error(w,
			"Failed to encode results",
			http.StatusInternalServerError)
		return
	}
}

func (rs *RestServer) SensorReadHandler(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("id")
	if device == "" {
		http.Error(w, "Device not given", http.StatusBadRequest)
		return
	}

	cachedData, found := rs.Cache.Get(device)
	if !found {
		http.Error(w, "Invalid device", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(true)
	if err := enc.Encode(cachedData); err != nil {
		log.Printf("Failed to encode results: %s", err.Error())
		http.Error(w,
			"Failed to encode results",
			http.StatusInternalServerError)
		return
	}

}

