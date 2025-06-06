// Package config provides configuration management for the MCP Memory Server,
// handling environment variables, YAML files, and runtime settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Qdrant   QdrantConfig   `json:"qdrant"`
	OpenAI   OpenAIConfig   `json:"openai"`
	Storage  StorageConfig  `json:"storage"`
	Chunking ChunkingConfig `json:"chunking"`
	Search   SearchConfig   `json:"search"`
	Logging  LoggingConfig  `json:"logging"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
}

// QdrantConfig represents Qdrant vector database configuration
type QdrantConfig struct {
	Host           string       `json:"host"`
	Port           int          `json:"port"`
	APIKey         string       `json:"-"` // Never serialize API key
	UseTLS         bool         `json:"use_tls"`
	Collection     string       `json:"collection"`
	Docker         DockerConfig `json:"docker"`
	HealthCheck    bool         `json:"health_check"`
	RetryAttempts  int          `json:"retry_attempts"`
	TimeoutSeconds int          `json:"timeout_seconds"`
}

// DockerConfig represents Docker-specific configuration
type DockerConfig struct {
	Enabled       bool   `json:"enabled"`
	ContainerName string `json:"container_name"`
	VolumePath    string `json:"volume_path"`
	Image         string `json:"image"`
}

// OpenAIConfig represents OpenAI API configuration
type OpenAIConfig struct {
	APIKey         string  `json:"-"` // Never serialize API key
	EmbeddingModel string  `json:"embedding_model"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	RequestTimeout int     `json:"request_timeout_seconds"`
	RateLimitRPM   int     `json:"rate_limit_rpm"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Provider       string                `json:"provider"`
	RetentionDays  int                   `json:"retention_days"`
	BackupEnabled  bool                  `json:"backup_enabled"`
	BackupInterval int                   `json:"backup_interval_hours"`
	Repositories   map[string]RepoConfig `json:"repositories"`
}

// RepoConfig represents repository-specific configuration
type RepoConfig struct {
	Enabled         bool     `json:"enabled"`
	Sensitivity     string   `json:"sensitivity"`
	ExcludePatterns []string `json:"exclude_patterns"`
	Tags            []string `json:"tags"`
}

// ChunkingConfig represents chunking algorithm configuration
type ChunkingConfig struct {
	Strategy              string  `json:"strategy"`
	MinContentLength      int     `json:"min_content_length"`
	MaxContentLength      int     `json:"max_content_length"`
	TodoCompletionTrigger bool    `json:"todo_completion_trigger"`
	FileChangeThreshold   int     `json:"file_change_threshold"`
	TimeThresholdMinutes  int     `json:"time_threshold_minutes"`
	SimilarityThreshold   float64 `json:"similarity_threshold"`
}

// SearchConfig represents search behavior configuration
type SearchConfig struct {
	DefaultMinRelevance      float64 `json:"default_min_relevance"`
	RelaxedMinRelevance      float64 `json:"relaxed_min_relevance"`
	BroadestMinRelevance     float64 `json:"broadest_min_relevance"`
	EnableProgressiveSearch  bool    `json:"enable_progressive_search"`
	EnableRepositoryFallback bool    `json:"enable_repository_fallback"`
	MaxRelatedRepos          int     `json:"max_related_repos"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	File       string `json:"file,omitempty"`
	MaxSize    int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age_days"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "localhost",
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Qdrant: QdrantConfig{
			Host:           "localhost",
			Port:           6334,
			UseTLS:         false,
			Collection:     "claude_memory",
			HealthCheck:    true,
			RetryAttempts:  3,
			TimeoutSeconds: 30,
			Docker: DockerConfig{
				Enabled:       true,
				ContainerName: "claude-memory-qdrant",
				VolumePath:    "./data/qdrant",
				Image:         "qdrant/qdrant:latest",
			},
		},
		OpenAI: OpenAIConfig{
			EmbeddingModel: "text-embedding-ada-002",
			MaxTokens:      8191,
			Temperature:    0.0,
			RequestTimeout: 60,
			RateLimitRPM:   60,
		},
		Storage: StorageConfig{
			Provider:       "qdrant",
			RetentionDays:  90,
			BackupEnabled:  false,
			BackupInterval: 24,
			Repositories:   make(map[string]RepoConfig),
		},
		Chunking: ChunkingConfig{
			Strategy:              "smart",
			MinContentLength:      50,
			MaxContentLength:      10000,
			TodoCompletionTrigger: true,
			FileChangeThreshold:   3,
			TimeThresholdMinutes:  10,
			SimilarityThreshold:   0.8,
		},
		Search: SearchConfig{
			DefaultMinRelevance:      0.5,
			RelaxedMinRelevance:      0.3,
			BroadestMinRelevance:     0.2,
			EnableProgressiveSearch:  true,
			EnableRepositoryFallback: true,
			MaxRelatedRepos:          3,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     30,
		},
	}
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	config := DefaultConfig()

	// Override with environment variables
	loadFromEnv(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	loadServerConfig(config)
	loadQdrantConfig(config)
	loadStorageAndOtherConfig(config)
	loadOpenAIConfig(config)
	loadDecayConfig(config)
	loadIntelligenceConfig(config)
	loadPerformanceConfig(config)
}

// loadServerConfig loads server configuration from environment
func loadServerConfig(config *Config) {
	// Server configuration
	if port := os.Getenv("MCP_MEMORY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if host := os.Getenv("MCP_MEMORY_HOST"); host != "" {
		config.Server.Host = host
	}

	// Server timeouts
	if readTimeout := os.Getenv("MCP_MEMORY_READ_TIMEOUT_SECONDS"); readTimeout != "" {
		if rt, err := strconv.Atoi(readTimeout); err == nil {
			config.Server.ReadTimeout = rt
		}
	}
	if writeTimeout := os.Getenv("MCP_MEMORY_WRITE_TIMEOUT_SECONDS"); writeTimeout != "" {
		if wt, err := strconv.Atoi(writeTimeout); err == nil {
			config.Server.WriteTimeout = wt
		}
	}
}

// loadQdrantConfig loads Qdrant configuration from environment
func loadQdrantConfig(config *Config) {
	loadQdrantBasicConfig(config)
	loadQdrantDockerConfig(config)
}

// loadQdrantBasicConfig loads basic Qdrant settings
func loadQdrantBasicConfig(config *Config) {
	loadQdrantConnectionSettings(config)
	loadQdrantServiceSettings(config)
}

// loadQdrantConnectionSettings loads host, port, API key, and TLS settings
func loadQdrantConnectionSettings(config *Config) {
	config.Qdrant.Host = getStringEnvWithFallback("MCP_MEMORY_QDRANT_HOST", "QDRANT_HOST", config.Qdrant.Host)
	config.Qdrant.Port = getIntEnvWithFallback("MCP_MEMORY_QDRANT_PORT", "QDRANT_PORT", config.Qdrant.Port)
	config.Qdrant.APIKey = getStringEnvWithFallback("MCP_MEMORY_QDRANT_API_KEY", "QDRANT_API_KEY", config.Qdrant.APIKey)
	config.Qdrant.UseTLS = getBoolEnvWithFallback("MCP_MEMORY_QDRANT_USE_TLS", "QDRANT_USE_TLS", config.Qdrant.UseTLS)
	config.Qdrant.Collection = getStringEnvWithFallback("MCP_MEMORY_QDRANT_COLLECTION", "QDRANT_COLLECTION", config.Qdrant.Collection)
}

// loadQdrantServiceSettings loads service-related settings like health check, retry, and timeout
func loadQdrantServiceSettings(config *Config) {
	config.Qdrant.HealthCheck = getBoolEnvWithDefault("MCP_MEMORY_QDRANT_HEALTH_CHECK", config.Qdrant.HealthCheck)
	config.Qdrant.RetryAttempts = getIntEnvWithDefault("MCP_MEMORY_QDRANT_RETRY_ATTEMPTS", config.Qdrant.RetryAttempts)
	config.Qdrant.TimeoutSeconds = getIntEnvWithDefault("MCP_MEMORY_QDRANT_TIMEOUT_SECONDS", config.Qdrant.TimeoutSeconds)
}

// getStringEnvWithFallback gets string environment variable with fallback to alternate key
func getStringEnvWithFallback(primaryKey, fallbackKey, defaultValue string) string {
	if value := os.Getenv(primaryKey); value != "" {
		return value
	}
	if value := os.Getenv(fallbackKey); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnvWithFallback gets integer environment variable with fallback to alternate key
func getIntEnvWithFallback(primaryKey, fallbackKey string, defaultValue int) int {
	if value := os.Getenv(primaryKey); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	if value := os.Getenv(fallbackKey); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getBoolEnvWithFallback gets boolean environment variable with fallback to alternate key
func getBoolEnvWithFallback(primaryKey, fallbackKey string, defaultValue bool) bool {
	if value := os.Getenv(primaryKey); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	if value := os.Getenv(fallbackKey); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getBoolEnvWithDefault gets boolean environment variable with default value
func getBoolEnvWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getIntEnvWithDefault gets integer environment variable with default value
func getIntEnvWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// loadQdrantDockerConfig loads Docker-related Qdrant settings
func loadQdrantDockerConfig(config *Config) {
	if dockerEnabled := os.Getenv("MCP_MEMORY_QDRANT_DOCKER_ENABLED"); dockerEnabled != "" {
		if de, err := strconv.ParseBool(dockerEnabled); err == nil {
			config.Qdrant.Docker.Enabled = de
		}
	}
	if containerName := os.Getenv("QDRANT_CONTAINER_NAME"); containerName != "" {
		config.Qdrant.Docker.ContainerName = containerName
	}
	if volumePath := os.Getenv("QDRANT_VOLUME_PATH"); volumePath != "" {
		config.Qdrant.Docker.VolumePath = volumePath
	}
	if image := os.Getenv("MCP_MEMORY_QDRANT_IMAGE"); image != "" {
		config.Qdrant.Docker.Image = image
	}
}

func loadStorageAndOtherConfig(config *Config) {
	loadStorageConfig(config)
	loadChunkingConfig(config)
	loadLoggingConfig(config)
}

// loadStorageConfig loads storage configuration from environment
func loadStorageConfig(config *Config) {
	if provider := os.Getenv("MCP_MEMORY_STORAGE_PROVIDER"); provider != "" {
		config.Storage.Provider = provider
	}
	if retention := os.Getenv("RETENTION_DAYS"); retention != "" {
		if r, err := strconv.Atoi(retention); err == nil {
			config.Storage.RetentionDays = r
		}
	}
	if backupEnabled := os.Getenv("MCP_MEMORY_BACKUP_ENABLED"); backupEnabled != "" {
		if be, err := strconv.ParseBool(backupEnabled); err == nil {
			config.Storage.BackupEnabled = be
		}
	}
	if backupInterval := os.Getenv("MCP_MEMORY_BACKUP_INTERVAL_HOURS"); backupInterval != "" {
		if bi, err := strconv.Atoi(backupInterval); err == nil {
			config.Storage.BackupInterval = bi
		}
	}
}

// loadChunkingConfig loads chunking configuration from environment
func loadChunkingConfig(config *Config) {
	if strategy := os.Getenv("MCP_MEMORY_CHUNKING_STRATEGY"); strategy != "" {
		config.Chunking.Strategy = strategy
	}
	if minLength := os.Getenv("MCP_MEMORY_CHUNKING_MIN_LENGTH"); minLength != "" {
		if ml, err := strconv.Atoi(minLength); err == nil {
			config.Chunking.MinContentLength = ml
		}
	}
	if maxLength := os.Getenv("MCP_MEMORY_CHUNKING_MAX_LENGTH"); maxLength != "" {
		if ml, err := strconv.Atoi(maxLength); err == nil {
			config.Chunking.MaxContentLength = ml
		}
	}
	if todoTrigger := os.Getenv("MCP_MEMORY_CHUNKING_TODO_TRIGGER"); todoTrigger != "" {
		if tt, err := strconv.ParseBool(todoTrigger); err == nil {
			config.Chunking.TodoCompletionTrigger = tt
		}
	}
}

// loadLoggingConfig loads logging configuration from environment
func loadLoggingConfig(config *Config) {
	if level := os.Getenv("MCP_MEMORY_LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("MCP_MEMORY_LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}
	if file := os.Getenv("MCP_MEMORY_LOG_FILE"); file != "" {
		config.Logging.File = file
	}
	if maxSize := os.Getenv("MCP_MEMORY_LOG_MAX_SIZE_MB"); maxSize != "" {
		if ms, err := strconv.Atoi(maxSize); err == nil {
			config.Logging.MaxSize = ms
		}
	}
	if maxBackups := os.Getenv("MCP_MEMORY_LOG_MAX_BACKUPS"); maxBackups != "" {
		if mb, err := strconv.Atoi(maxBackups); err == nil {
			config.Logging.MaxBackups = mb
		}
	}
	if maxAge := os.Getenv("MCP_MEMORY_LOG_MAX_AGE_DAYS"); maxAge != "" {
		if ma, err := strconv.Atoi(maxAge); err == nil {
			config.Logging.MaxAge = ma
		}
	}
}

// loadOpenAIConfig loads OpenAI configuration from environment
func loadOpenAIConfig(config *Config) {
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_EMBEDDING_MODEL"); model != "" {
		config.OpenAI.EmbeddingModel = model
	}
	if maxTokens := os.Getenv("MCP_MEMORY_OPENAI_MAX_TOKENS"); maxTokens != "" {
		if mt, err := strconv.Atoi(maxTokens); err == nil {
			config.OpenAI.MaxTokens = mt
		}
	}
	if temperature := os.Getenv("MCP_MEMORY_OPENAI_TEMPERATURE"); temperature != "" {
		if temp, err := strconv.ParseFloat(temperature, 64); err == nil {
			config.OpenAI.Temperature = temp
		}
	}
	if requestTimeout := os.Getenv("MCP_MEMORY_OPENAI_REQUEST_TIMEOUT_SECONDS"); requestTimeout != "" {
		if rt, err := strconv.Atoi(requestTimeout); err == nil {
			config.OpenAI.RequestTimeout = rt
		}
	}
	if rateLimitRPM := os.Getenv("MCP_MEMORY_OPENAI_RATE_LIMIT_RPM"); rateLimitRPM != "" {
		if rl, err := strconv.Atoi(rateLimitRPM); err == nil {
			config.OpenAI.RateLimitRPM = rl
		}
	}
}

// loadDecayConfig loads decay configuration from environment
func loadDecayConfig(_ *Config) {
	// Add decay config loading if needed
}

// loadIntelligenceConfig loads intelligence configuration from environment
func loadIntelligenceConfig(_ *Config) {
	// Add intelligence config loading if needed
}

// loadPerformanceConfig loads performance configuration from environment
func loadPerformanceConfig(config *Config) {
	// Add performance config loading if needed
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.validateServerConfig(); err != nil {
		return err
	}

	if err := c.validateQdrantConfig(); err != nil {
		return err
	}

	if err := c.validateOpenAIConfig(); err != nil {
		return err
	}

	if err := c.validateStorageConfig(); err != nil {
		return err
	}

	if err := c.validateChunkingConfig(); err != nil {
		return err
	}

	return nil
}

// validateServerConfig validates server configuration settings
func (c *Config) validateServerConfig() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.Host == "" {
		return errors.New("server host cannot be empty")
	}
	return nil
}

// validateQdrantConfig validates Qdrant vector database configuration
func (c *Config) validateQdrantConfig() error {
	if c.Qdrant.Host == "" {
		return errors.New("qdrant host cannot be empty")
	}
	if c.Qdrant.Port <= 0 {
		return errors.New("qdrant port must be greater than 0")
	}
	if c.Qdrant.Collection == "" {
		return errors.New("qdrant collection cannot be empty")
	}
	if c.Qdrant.Docker.Enabled && c.Qdrant.Docker.ContainerName == "" {
		return errors.New("docker container name cannot be empty when docker is enabled")
	}
	return nil
}

// validateOpenAIConfig validates OpenAI API configuration
func (c *Config) validateOpenAIConfig() error {
	if c.OpenAI.APIKey == "" {
		return errors.New("OpenAI API key is required")
	}
	if c.OpenAI.EmbeddingModel == "" {
		return errors.New("OpenAI embedding model cannot be empty")
	}
	return nil
}

// validateStorageConfig validates storage configuration settings
func (c *Config) validateStorageConfig() error {
	if c.Storage.RetentionDays <= 0 {
		return errors.New("retention days must be positive")
	}
	return nil
}

// validateChunkingConfig validates chunking algorithm configuration
func (c *Config) validateChunkingConfig() error {
	if c.Chunking.MinContentLength <= 0 {
		return errors.New("min content length must be positive")
	}
	if c.Chunking.MaxContentLength <= c.Chunking.MinContentLength {
		return errors.New("max content length must be greater than min content length")
	}
	if c.Chunking.SimilarityThreshold < 0 || c.Chunking.SimilarityThreshold > 1 {
		return errors.New("similarity threshold must be between 0 and 1")
	}
	return nil
}

// GetDataDir returns the data directory path, creating it if necessary
func (c *Config) GetDataDir() (string, error) {
	dataDir := c.Qdrant.Docker.VolumePath
	if dataDir == "" {
		dataDir = "./data"
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for data directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0o750); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return absPath, nil
}

// GetRepoConfig returns configuration for a specific repository
func (c *Config) GetRepoConfig(repository string) RepoConfig {
	if repoConfig, exists := c.Storage.Repositories[repository]; exists {
		return repoConfig
	}

	// Return default repo config
	return RepoConfig{
		Enabled:         true,
		Sensitivity:     "normal",
		ExcludePatterns: []string{"*.env", "*.key", "*.pem", "*.p12"},
		Tags:            []string{},
	}
}

// SetRepoConfig sets configuration for a specific repository
func (c *Config) SetRepoConfig(repository string, config RepoConfig) {
	if c.Storage.Repositories == nil {
		c.Storage.Repositories = make(map[string]RepoConfig)
	}
	c.Storage.Repositories[repository] = config
}

// IsRepositoryEnabled checks if a repository is enabled for memory storage
func (c *Config) IsRepositoryEnabled(repository string) bool {
	repoConfig := c.GetRepoConfig(repository)
	return repoConfig.Enabled
}
