package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Request metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_spikes_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	HTTPRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"method", "endpoint"},
	)

	// Fibonacci computation metrics
	FibonacciComputations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_fibonacci_computations_total",
			Help: "Total number of Fibonacci computations",
		},
		[]string{"input_value"},
	)

	FibonacciComputationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_spikes_fibonacci_computation_duration_seconds",
			Help:    "Duration of Fibonacci computations in seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"input_value"},
	)

	FibonacciComputationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_fibonacci_computation_errors_total",
			Help: "Total number of Fibonacci computation errors",
		},
		[]string{"error_type"},
	)

	// Kafka metrics
	KafkaMessagesProduced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_kafka_messages_produced_total",
			Help: "Total number of Kafka messages produced",
		},
		[]string{"topic", "partition"},
	)

	KafkaMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed",
		},
		[]string{"topic", "partition"},
	)

	KafkaProduceErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_kafka_produce_errors_total",
			Help: "Total number of Kafka produce errors",
		},
		[]string{"topic", "error_type"},
	)

	KafkaConsumeErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_kafka_consume_errors_total",
			Help: "Total number of Kafka consume errors",
		},
		[]string{"topic", "error_type"},
	)

	KafkaMessageSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_spikes_kafka_message_size_bytes",
			Help:    "Size of Kafka messages in bytes",
			Buckets: []float64{64, 256, 1024, 4096, 16384, 65536, 262144},
		},
		[]string{"topic", "direction"}, // direction: "produce" or "consume"
	)

	KafkaPartitionLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_kafka_partition_lag",
			Help: "Current lag for Kafka partitions",
		},
		[]string{"topic", "partition", "consumer_group"},
	)

	// Configuration metrics
	ConfigReloads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_config_reloads_total",
			Help: "Total number of configuration reloads",
		},
		[]string{"status"}, // status: "success" or "error"
	)

	ConfigFeatureFlags = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_config_feature_flags",
			Help: "Current state of feature flags (1=enabled, 0=disabled)",
		},
		[]string{"feature_name"},
	)

	// Database metrics
	DatabaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_database_connections",
			Help: "Number of active database connections",
		},
		[]string{"database_type"}, // mysql, postgres
	)

	DatabaseOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"database_type", "operation", "status"}, // operation: select, insert, update, delete
	)

	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_spikes_database_operation_duration_seconds",
			Help:    "Duration of database operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"database_type", "operation"},
	)

	// Application health metrics
	ApplicationInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_application_info",
			Help: "Application information (always 1)",
		},
		[]string{"version", "go_version"},
	)

	ApplicationUptime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_application_uptime_seconds",
			Help: "Application uptime in seconds",
		},
		[]string{},
	)

	// Worker/Runner metrics
	ActiveJobs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_spikes_active_jobs",
			Help: "Number of currently active jobs",
		},
		[]string{"job_type"},
	)

	JobExecutions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_spikes_job_executions_total",
			Help: "Total number of job executions",
		},
		[]string{"job_type", "status"}, // status: success, failure, timeout
	)

	JobExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_spikes_job_execution_duration_seconds",
			Help:    "Duration of job executions in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 300},
		},
		[]string{"job_type"},
	)
)

// RecordApplicationInfo records application metadata
func RecordApplicationInfo(version, goVersion string) {
	ApplicationInfo.WithLabelValues(version, goVersion).Set(1)
}

// UpdateApplicationUptime updates the application uptime metric
func UpdateApplicationUptime(seconds float64) {
	ApplicationUptime.WithLabelValues().Set(seconds)
}