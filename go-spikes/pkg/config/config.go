package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	k "github.com/infra-bed/go-spikes/pkg/config/kafka"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const DefaultLogBatchSize = 10000

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Kafka    k.KafkaConfig  `mapstructure:"kafka"`
	Database DatabaseConfig `mapstructure:"database"`
	Features FeatureFlags   `mapstructure:"features"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Tests    TestsConfig    `mapstructure:"tests"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`
}

type DatabaseConfig struct {
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Postgres PostgresConfig `mapstructure:"postgres"`
}

type MySQLConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	User            string        `mapstructure:"user"`
	MaxConnections  int           `mapstructure:"maxConnections"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
}

type PostgresConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	User            string        `mapstructure:"user"`
	SSLMode         string        `mapstructure:"sslMode"`
	MaxConnections  int           `mapstructure:"maxConnections"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
}

type FeatureFlags struct {
	EnableProfiling   bool            `mapstructure:"enableProfiling"`
	EnableTracing     bool            `mapstructure:"enableTracing"`
	EnableMetrics     bool            `mapstructure:"enableMetrics"`
	LogLevel          string          `mapstructure:"logLevel"`
	ExperimentalFlags map[string]bool `mapstructure:"experimental"`
}

type MetricsConfig struct {
	ScrapeInterval   time.Duration     `mapstructure:"scrapeInterval"`
	HistogramBuckets []float64         `mapstructure:"histogramBuckets"`
	Labels           map[string]string `mapstructure:"labels"`
}

type TestsConfig struct {
	EntityRepoConfig k.EntityRepoConfig `mapstructure:"entityRepo"`
}

type ConfigManager struct {
	mu              sync.RWMutex
	config          *Config
	v               *viper.Viper
	changeCallbacks []func(*Config)
}

func NewConfigManager(configPath string) (*ConfigManager, error) {
	cm := &ConfigManager{
		config:          &Config{},
		v:               viper.New(),
		changeCallbacks: make([]func(*Config), 0),
	}

	cm.v.SetConfigFile(configPath)
	cm.v.SetConfigType("yaml")

	cm.setDefaults()

	if err := cm.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Warn().Msg("Config file not found, using defaults")
	}

	if err := cm.v.Unmarshal(cm.config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Start custom watcher for Kubernetes ConfigMap updates
	go cm.watchConfigFile(configPath)

	return cm, nil
}

func (cm *ConfigManager) setDefaults() {
	cm.setDefaultsForViper(cm.v)
}

func (cm *ConfigManager) watchConfigFile(configPath string) {
	// Kubernetes mounts ConfigMaps using symlinks
	// We need to watch the directory for changes, not just the file
	dir := filepath.Dir(configPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file watcher")
		return
	}
	defer func(watcher *fsnotify.Watcher) {
		err := watcher.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close file watcher")
		} else {
			log.Debug().Msg("File watcher closed successfully")
		}
	}(watcher)

	// Watch the directory containing the config file
	if err := watcher.Add(dir); err != nil {
		log.Error().Err(err).Str("dir", dir).Msg("Failed to watch config directory")
		return
	}

	log.Info().Str("path", configPath).Str("dir", dir).Msg("Watching config file for changes")

	// Also try to watch the file directly (for non-Kubernetes environments)
	if err := watcher.Add(configPath); err != nil {
		log.Debug().Err(err).Str("file", configPath).Msg("Could not watch file directly, watching directory only")
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			log.Debug().
				Str("event", event.String()).
				Str("name", event.Name).
				Str("op", event.Op.String()).
				Msg("File watcher event")

			// Kubernetes updates ConfigMaps by creating new files and updating symlinks
			// We need to detect various types of changes
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				// Check if the event is related to our config file or the data directory
				if event.Name == configPath || filepath.Base(event.Name) == "..data" || event.Name == filepath.Join(dir, "..data") {
					log.Info().Str("event", event.String()).Msg("Config file change detected")

					// Small delay to ensure the file write is complete
					time.Sleep(100 * time.Millisecond)

					// Re-read the config
					if err := cm.reloadFromFile(configPath); err != nil {
						log.Error().Err(err).Msg("Failed to reload config")
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("File watcher error")
		}
	}
}

func (cm *ConfigManager) reloadFromFile(configPath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create a new viper instance to read the updated file
	newViper := viper.New()
	newViper.SetConfigFile(configPath)
	newViper.SetConfigType("yaml")

	// Copy defaults
	cm.setDefaultsForViper(newViper)

	// Read the config file
	if err := newViper.ReadInConfig(); err != nil {
		// Check if file exists - it might be in the middle of being updated
		if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
			log.Debug().Msg("Config file temporarily missing, likely being updated")
			return nil
		}
		return fmt.Errorf("error reading updated config: %w", err)
	}

	// Unmarshal into new config
	newConfig := &Config{}
	if err := newViper.Unmarshal(newConfig); err != nil {
		return fmt.Errorf("error unmarshaling updated config: %w", err)
	}

	// Update the config
	oldConfig := cm.config
	cm.config = newConfig
	cm.v = newViper

	log.Info().Msg("Configuration reloaded successfully from file")

	// Notify callbacks
	for _, callback := range cm.changeCallbacks {
		go func(cb func(*Config)) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("Panic in config change callback")
				}
			}()
			cb(newConfig)
		}(callback)
	}

	log.Debug().
		Interface("old", oldConfig).
		Interface("new", newConfig).
		Msg("Configuration updated")

	return nil
}

func (cm *ConfigManager) setDefaultsForViper(v *viper.Viper) {
	v.SetDefault("server.port", 8888)
	v.SetDefault("server.readTimeout", "30s")
	v.SetDefault("server.writeTimeout", "30s")
	v.SetDefault("server.idleTimeout", "120s")

	v.SetDefault("kafka.brokers", []string{"kafka-cluster-kafka-bootstrap.kafka:9092"})
	v.SetDefault("kafka.topic", "test-topic")
	v.SetDefault("kafka.consumerGroup", "go-spikes-consumer")
	v.SetDefault("kafka.producer.batchSize", 100)
	v.SetDefault("kafka.producer.batchTimeout", "1s")
	v.SetDefault("kafka.producer.compressionType", "snappy")
	v.SetDefault("kafka.producer.maxRetries", 3)
	v.SetDefault("kafka.consumer.sessionTimeout", "10s")
	v.SetDefault("kafka.consumer.heartbeatInterval", "3s")
	v.SetDefault("kafka.consumer.maxPollRecords", 500)
	v.SetDefault("kafka.consumer.autoOffsetReset", "latest")

	v.SetDefault("database.mysql.enabled", false)
	v.SetDefault("database.mysql.host", "mycluster-router.default")
	v.SetDefault("database.mysql.port", 6446)
	v.SetDefault("database.mysql.database", "test_db")
	v.SetDefault("database.mysql.user", "root")
	v.SetDefault("database.mysql.maxConnections", 25)
	v.SetDefault("database.mysql.maxIdleConns", 5)
	v.SetDefault("database.mysql.connMaxLifetime", "5m")

	v.SetDefault("database.postgres.enabled", false)
	v.SetDefault("database.postgres.host", "postgres-cluster-rw.default")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.database", "myapp")
	v.SetDefault("database.postgres.user", "app")
	v.SetDefault("database.postgres.sslMode", "disable")
	v.SetDefault("database.postgres.maxConnections", 25)
	v.SetDefault("database.postgres.maxIdleConns", 5)
	v.SetDefault("database.postgres.connMaxLifetime", "5m")

	v.SetDefault("features.enableProfiling", true)
	v.SetDefault("features.enableTracing", true)
	v.SetDefault("features.enableMetrics", true)
	v.SetDefault("features.enableDebugLogging", false)
	v.SetDefault("features.logLevel", "info")
	v.SetDefault("features.experimental", map[string]bool{})

	v.SetDefault("metrics.scrapeInterval", "10s")
	v.SetDefault("metrics.histogramBuckets", []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10})
	v.SetDefault("metrics.labels", map[string]string{})
}

func (cm *ConfigManager) reload() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	newConfig := &Config{}
	if err := cm.v.Unmarshal(newConfig); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal new config, keeping old config")
		return
	}

	oldConfig := cm.config
	cm.config = newConfig

	log.Info().Msg("Configuration reloaded successfully")

	for _, callback := range cm.changeCallbacks {
		go func(cb func(*Config)) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("Panic in config change callback")
				}
			}()
			cb(newConfig)
		}(callback)
	}

	log.Debug().
		Interface("old", oldConfig).
		Interface("new", newConfig).
		Msg("Configuration updated")
}

func (cm *ConfigManager) Get() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) OnChange(callback func(*Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.changeCallbacks = append(cm.changeCallbacks, callback)
}

func (cm *ConfigManager) GetServer() ServerConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Server
}

func (cm *ConfigManager) GetKafka() k.KafkaConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Kafka
}

func (cm *ConfigManager) GetDatabase() DatabaseConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Database
}

func (cm *ConfigManager) GetFeatures() FeatureFlags {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Features
}

func (cm *ConfigManager) GetMetrics() MetricsConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Metrics
}

func (cm *ConfigManager) GetTests() TestsConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.Tests
}

func (cm *ConfigManager) IsFeatureEnabled(feature string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if val, ok := cm.config.Features.ExperimentalFlags[feature]; ok {
		return val
	}
	return false
}
