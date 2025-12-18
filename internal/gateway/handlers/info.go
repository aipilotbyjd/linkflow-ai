package handlers

import (
	"encoding/json"
	"net/http"
)

func GatewayInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service": "API Gateway",
		"version": "1.0.0",
		"status":  "running",
		"services": map[string]string{
			"auth":         "http://localhost:8001",
			"user":         "http://localhost:8002",
			"execution":    "http://localhost:8003",
			"workflow":     "http://localhost:8004",
			"node":         "http://localhost:8005",
			"schedule":     "http://localhost:8006",
			"webhook":      "http://localhost:8007",
			"notification": "http://localhost:8008",
			"analytics":    "http://localhost:8009",
			"search":       "http://localhost:8010",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
