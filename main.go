package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	libraryOccupancy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tum_library_occupancy_percent",
			Help: "Current occupancy of TUM branch libraries in percent.",
		},
		[]string{"library"},
	)
)

func init() {
	prometheus.MustRegister(libraryOccupancy)
}

type usageResponse struct {
	Usage float64 `json:"usage"`
}

var locations = map[string]string{
	"Stammgelaende":      "Stammgelände",
	"Maschinenwesen":    "Maschinenwesen",
	"Physik":            "Physik",
	"Sport":             "Sport- & Gesundheitswissenschaften",
	"Weihenstephan":     "Weihenstephan",
	"Chemie":            "Chemie",
	"Straubing":         "Straubing",
}

func scrapeOccupancy() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for backendName, labelName := range locations {
		payload := map[string]string{
			"location": backendName,
		}

		jsonPayload, _ := json.Marshal(payload)

		form := url.Values{}
		form.Set("loadGraphiteData", string(jsonPayload))

		req, err := http.NewRequest(
			http.MethodPost,
			"https://auslastungsanzeige.ub.tum.de/backend/FrontController.php",
			bytes.NewBufferString(form.Encode()),
		)
		if err != nil {
			log.Println("request error:", err)
			continue
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")

		resp, err := client.Do(req)
		if err != nil {
			log.Println("http error:", err)
			continue
		}

		var result usageResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if err != nil {
			log.Println("decode error:", err)
			continue
		}

		usage := result.Usage
		if usage < 0 {
			usage = 0
		}
		log.Printf("%s: %.0f%%\n", labelName, usage)
		libraryOccupancy.WithLabelValues(labelName).Set(usage)
	}
}

func main() {
	go func() {
		for {
			log.Println("Scraping TUM Library occupancy...")
			scrapeOccupancy()
			time.Sleep(1 * time.Minute)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Exporter running on :8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
