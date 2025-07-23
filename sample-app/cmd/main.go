package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlptracegrpc.WithInsecure(),
	))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("sample-app"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

type Response struct {
	Message    string      `json:"message"`
	Duration   string      `json:"duration,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Iterations int         `json:"iterations,omitempty"`
}

func main() {
	ctx := context.Background()

	// Initialize tracing
	tp, err := initTracer(ctx)
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	} else {
		defer tp.Shutdown(ctx)
		// Wrap tracer provider for Pyroscope integration
		otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tp))
	}

	// Note: We're using Alloy to scrape pprof endpoints instead of pushing directly
	// This allows for better integration with the Grafana stack

	// Start pprof server on separate port
	pprofPort := os.Getenv("PPROF_PORT")
	if pprofPort == "" {
		pprofPort = "6060"
	}
	go func() {
		log.Printf("Starting pprof server on port %s", pprofPort)
		log.Println(http.ListenAndServe(":"+pprofPort, nil))
	}()

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("sample-app"))

	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// CPU-intensive endpoints
	r.HandleFunc("/cpu/fibonacci/{n}", fibonacciHandler).Methods("GET")
	r.HandleFunc("/cpu/prime/{n}", primeHandler).Methods("GET")
	r.HandleFunc("/cpu/hash/{iterations}", hashHandler).Methods("GET")
	r.HandleFunc("/cpu/sort/{size}", sortHandler).Methods("GET")
	r.HandleFunc("/cpu/matrix/{size}", matrixHandler).Methods("GET")

	// Memory-intensive endpoints
	r.HandleFunc("/memory/allocate/{mb}", memoryHandler).Methods("GET")

	// Mixed workload
	r.HandleFunc("/workload/mixed", mixedWorkloadHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting sample-app on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Response{Message: "healthy"})
}

func fibonacciHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	n, err := strconv.Atoi(vars["n"])
	if err != nil || n < 1 || n > 45 {
		http.Error(w, "Invalid number (1-45)", http.StatusBadRequest)
		return
	}

	start := time.Now()
	result := fibonacci(n)
	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Fibonacci of %d", n),
		Duration: duration.String(),
		Result:   result,
	})
}

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func primeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	n, err := strconv.Atoi(vars["n"])
	if err != nil || n < 1 || n > 1000000 {
		http.Error(w, "Invalid number (1-1000000)", http.StatusBadRequest)
		return
	}

	start := time.Now()
	primes := sieveOfEratosthenes(n)
	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Prime numbers up to %d", n),
		Duration: duration.String(),
		Result:   len(primes),
	})
}

func sieveOfEratosthenes(n int) []int {
	isPrime := make([]bool, n+1)
	for i := 2; i <= n; i++ {
		isPrime[i] = true
	}

	for i := 2; i*i <= n; i++ {
		if isPrime[i] {
			for j := i * i; j <= n; j += i {
				isPrime[j] = false
			}
		}
	}

	primes := []int{}
	for i := 2; i <= n; i++ {
		if isPrime[i] {
			primes = append(primes, i)
		}
	}
	return primes
}

func hashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iterations, err := strconv.Atoi(vars["iterations"])
	if err != nil || iterations < 1 || iterations > 1000000 {
		http.Error(w, "Invalid iterations (1-1000000)", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data := []byte("OpenTelemetry eBPF Profiling Test")
	var hash [32]byte

	for i := 0; i < iterations; i++ {
		hash = sha256.Sum256(data)
		data = hash[:]
	}

	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:    fmt.Sprintf("Computed %d SHA256 hashes", iterations),
		Duration:   duration.String(),
		Result:     fmt.Sprintf("%x", hash),
		Iterations: iterations,
	})
}

func sortHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	size, err := strconv.Atoi(vars["size"])
	if err != nil || size < 1 || size > 1000000 {
		http.Error(w, "Invalid size (1-1000000)", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Generate random data
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = size - i
	}

	// Sort the data
	sort.Ints(data)

	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Sorted %d integers", size),
		Duration: duration.String(),
	})
}

func matrixHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	size, err := strconv.Atoi(vars["size"])
	if err != nil || size < 1 || size > 500 {
		http.Error(w, "Invalid size (1-500)", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Create two matrices
	a := make([][]float64, size)
	b := make([][]float64, size)
	c := make([][]float64, size)

	for i := 0; i < size; i++ {
		a[i] = make([]float64, size)
		b[i] = make([]float64, size)
		c[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			a[i][j] = float64(i + j)
			b[i][j] = float64(i - j)
		}
	}

	// Matrix multiplication
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			for k := 0; k < size; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}

	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Multiplied %dx%d matrices", size, size),
		Duration: duration.String(),
		Result:   c[0][0], // Return first element as sample
	})
}

func memoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mb, err := strconv.Atoi(vars["mb"])
	if err != nil || mb < 1 || mb > 100 {
		http.Error(w, "Invalid size (1-100 MB)", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Allocate memory
	data := make([]byte, mb*1024*1024)

	// Touch all pages to ensure allocation
	for i := 0; i < len(data); i += 4096 {
		data[i] = byte(i % 256)
	}

	// Force a GC to see memory usage
	runtime.GC()

	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Allocated %d MB", mb),
		Duration: duration.String(),
	})
}

func mixedWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// CPU work: Fibonacci
	fib := fibonacci(35)

	// Memory allocation
	data := make([]byte, 10*1024*1024) // 10MB
	for i := 0; i < len(data); i += 4096 {
		data[i] = byte(i % 256)
	}

	// More CPU work: Sorting
	nums := make([]int, 100000)
	for i := 0; i < len(nums); i++ {
		nums[i] = len(nums) - i
	}
	sort.Ints(nums)

	// Hash computation
	hash := sha256.Sum256(data)

	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  "Completed mixed workload",
		Duration: duration.String(),
		Result: map[string]interface{}{
			"fibonacci": fib,
			"hash":      fmt.Sprintf("%x", hash[:8]),
			"sorted":    nums[0],
		},
	})
}
