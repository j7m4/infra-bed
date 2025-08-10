package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/infra-bed/go-spikes/pkg/fibonacci"
	"github.com/infra-bed/go-spikes/pkg/logger"
)

func Fibonacci(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.Ctx(ctx)

	vars := mux.Vars(r)
	n, err := strconv.Atoi(vars["n"])
	if err != nil || n < 1 || n > 45 {
		log.Warn().Int("n", n).Err(err).Msg("Invalid fibonacci number requested")
		http.Error(w, "Invalid number (1-45)", http.StatusBadRequest)
		return
	}

	start := time.Now()
	result := fibonacci.DoFibonacci(n)
	duration := time.Since(start)

	log.Info().
		Int("n", n).
		Int("result", result).
		Dur("duration", duration).
		Msg("Fibonacci calculated")

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Fibonacci of %d", n),
		Duration: duration.String(),
		Result:   result,
	})
}
