package handler

import (
	"encoding/json"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Response{Message: "healthy"})
}
