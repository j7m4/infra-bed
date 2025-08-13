package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/infra-bed/go-spikes/pkg/config"
	"github.com/infra-bed/go-spikes/pkg/logger"
)

var configManager *config.ConfigManager

func SetConfigManager(cm *config.ConfigManager) {
	configManager = cm
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()

	if configManager == nil {
		http.Error(w, "Configuration manager not initialized", http.StatusInternalServerError)
		return
	}

	cfg := configManager.Get()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		log.Error().Err(err).Msg("Failed to encode config")
		http.Error(w, "Failed to encode configuration", http.StatusInternalServerError)
		return
	}
}

func CheckFeature(w http.ResponseWriter, r *http.Request) {
	log := logger.Get()
	vars := mux.Vars(r)
	feature := vars["feature"]

	if configManager == nil {
		http.Error(w, "Configuration manager not initialized", http.StatusInternalServerError)
		return
	}

	enabled := false
	response := map[string]interface{}{
		"feature": feature,
	}

	cfg := configManager.GetFeatures()

	switch feature {
	case "profiling":
		enabled = cfg.EnableProfiling
	case "tracing":
		enabled = cfg.EnableTracing
	case "metrics":
		enabled = cfg.EnableMetrics
	case "debug":
		enabled = cfg.EnableDebugLogging
	default:
		enabled = configManager.IsFeatureEnabled(feature)
	}

	response["enabled"] = enabled

	log.Info().
		Str("feature", feature).
		Bool("enabled", enabled).
		Msg("Feature flag checked")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}