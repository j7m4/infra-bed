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
	"github.com/infra-bed/go-spikes/pkg/metrics"
)

func Fibonacci(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.Ctx(ctx)

	vars := mux.Vars(r)
	n, err := strconv.Atoi(vars["n"])
	if err != nil || n < 1 || n > 45 {
		log.Warn().Int("n", n).Err(err).Msg("Invalid fibonacci number requested")
		metrics.FibonacciComputationErrors.WithLabelValues("invalid_input").Inc()
		http.Error(w, "Invalid number (1-45)", http.StatusBadRequest)
		return
	}

	inputValue := strconv.Itoa(n)
	start := time.Now()
	metrics.FibonacciComputations.WithLabelValues(inputValue).Inc()
	
	result := fibonacci.DoFibonacciWithContext(ctx, n)
	duration := time.Since(start)
	
	metrics.FibonacciComputationDuration.WithLabelValues(inputValue).Observe(duration.Seconds())

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
