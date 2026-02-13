package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration from environment variables.
type Config struct {
	// Redis (broker - streams)
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisTLS      bool

	RedisIPAddress   string // general cache
	InstFBRedisHost  string
	InstFBRedisPort  string
	BackendRedisHost string
	BackendRedisPort string

	PublisherInputStream        string
	PublisherConsumerGroup      string
	PublisherConsumerName       string
	PublisherOutputStreamPrefix string
	PublisherDeadletterStream   string
	PublisherStreamMaxLen       int64
	PublisherClaimMinIdle        time.Duration

	OverrideOutput      bool
	OverrideRoutingKey  string

	// MongoDB
	MongoDBURI      string
	MongoDBHost     string
	MongoDBPort     string
	MongoDBUser     string
	MongoDBPassword string
	MongoDBDBName   string
	MongoDBTLS      bool
	MongoDBOptions  string

	// Cassandra
	CassandraServer1       string
	CassandraServer2       string
	CassandraDefaultKeyspace string

	// MinIO
	MinIOEndpoint        string
	MinIOAccessKey       string
	MinIOSecretKey       string
	MinIORegion          string
	MinIOUseSSL          bool
	MinIOWhatsAppBucket  string
	MinIOEngagementBucket string

	// HTTP APIs
	DialectsAIAPI              string
	ThemesAIAPI                string
	SubThemesAIAPI             string
	NetSentimentInterpretationAPI string
	ChatsTopKeywords           string
	CDPProfileAPI              string
	OverallSentimentURL        string
	AIAPIsTimeout             time.Duration
	BackendAPIURL              string
	LiveServiceURL             string

	// Behavior
	EventTimeout     time.Duration
	MaxTries         int
	CrashSleep       time.Duration
	ForceDisableInactivity bool
	ServerEnvironment string
	TotalStaciPosts  int64

	// Whitelabel
	WhitelabelProjectNameEN   string
	WhitelabelAIProductNameEN string
	WhitelabelProjectNameAR   string
	WhitelabelAIProductNameAR string
	WhitelabelPoweredBy       string
	WhitelabelLegalName       string
	WhitelabelPoweredBySentence string
	WhitelabelDisclaimerAR    string
	WhitelabelContactEmail    string
	WhitelabelAssumeResponsibility bool
	WhitelabelAIProductPPTLogo string

	// Observability
	OTELServiceName         string
	OTELTracesExporter      string
	OTELMetricsExporter     string
	OTELLogsExporter        string
	OTELExporterOTLPEndpoint string
	LogFmt                 bool
	NoSlack                bool

	// HTTP server
	HTTPPort string
}

// Load reads configuration from the environment. Returns error if required keys are missing.
func Load() (*Config, error) {
	c := &Config{}

	// Redis broker (default to backend redis if unset)
	c.RedisHost = getEnv("APP_REDIS_HOST", getEnv("APP_BACKEND_REDIS_SERVER", "localhost"))
	c.RedisPort = getEnv("APP_REDIS_PORT", getEnv("APP_BACKEND_REDIS_PORT", "6379"))
	c.RedisPassword = os.Getenv("APP_REDIS_PASSWORD")
	c.RedisTLS = parseBool(os.Getenv("APP_REDIS_TLS"))

	c.RedisIPAddress = getEnv("APP_REDIS_IP_ADDRESS", c.RedisHost)
	c.InstFBRedisHost = getEnv("APP_INST_FB_REDIS_IP_ADDRESS", c.RedisHost)
	c.InstFBRedisPort = getEnv("APP_INST_FB_REDIS_PORT", c.RedisPort)
	c.BackendRedisHost = getEnv("APP_BACKEND_REDIS_SERVER", c.RedisHost)
	c.BackendRedisPort = getEnv("APP_BACKEND_REDIS_PORT", c.RedisPort)

	c.PublisherInputStream = os.Getenv("APP_PUBLISHER_INPUT_STREAM")
	c.PublisherConsumerGroup = os.Getenv("APP_PUBLISHER_CONSUMER_GROUP")
	c.PublisherConsumerName = os.Getenv("APP_PUBLISHER_CONSUMER_NAME")
	c.PublisherOutputStreamPrefix = os.Getenv("APP_PUBLISHER_OUTPUT_STREAM_PREFIX")
	c.PublisherDeadletterStream = os.Getenv("APP_PUBLISHER_DEADLETTER_STREAM")
	c.PublisherStreamMaxLen = parseInt64(os.Getenv("APP_PUBLISHER_STREAM_MAXLEN"), 0)
	c.PublisherClaimMinIdle = time.Duration(parseInt64(os.Getenv("APP_PUBLISHER_CLAIM_MIN_IDLE"), 300000)) * time.Millisecond

	c.OverrideOutput = parseBool(os.Getenv("APP_GO_DISPATCHER_OVERRIDE"))
	c.OverrideRoutingKey = os.Getenv("APP_GO_DISPATCHER_OVERRIDE_ROUTING_KEY")

	// MongoDB
	c.MongoDBURI = os.Getenv("APP_MONGODB_URI")
	c.MongoDBHost = getEnv("APP_MONGODB_HOST", "localhost")
	c.MongoDBPort = getEnv("APP_MONGODB_PORT", "27017")
	c.MongoDBUser = os.Getenv("APP_MONGODB_USER")
	c.MongoDBPassword = os.Getenv("APP_MONGODB_PASSWORD")
	c.MongoDBDBName = os.Getenv("APP_MONGODB_DBNAME")
	c.MongoDBTLS = parseBool(os.Getenv("APP_MONGODB_TLS"))
	c.MongoDBOptions = os.Getenv("APP_MONGODB_OPTIONS")

	// Cassandra
	c.CassandraServer1 = os.Getenv("APP_CASSANDRA_SERVER_1")
	c.CassandraServer2 = os.Getenv("APP_CASSANDRA_SERVER_2")
	c.CassandraDefaultKeyspace = os.Getenv("APP_CASSANDRA_DEFAULT_KEYSPACE")

	// MinIO
	c.MinIOEndpoint = os.Getenv("APP_MINIO_ENDPOINT")
	c.MinIOAccessKey = os.Getenv("APP_MINIO_ACCESS_KEY")
	c.MinIOSecretKey = os.Getenv("APP_MINIO_SECRET_KEY")
	c.MinIORegion = getEnv("APP_MINIO_REGION", "us-east-1")
	c.MinIOUseSSL = parseBool(os.Getenv("APP_MINIO_USE_SSL"))
	c.MinIOWhatsAppBucket = os.Getenv("APP_MINIO_WHATSAPP_BUCKET")
	c.MinIOEngagementBucket = os.Getenv("APP_MINIO_ENGAGEMENT_BUCKET")

	// HTTP APIs
	c.DialectsAIAPI = os.Getenv("APP_DIALECTS_AI_API")
	c.ThemesAIAPI = os.Getenv("APP_THEMES_AI_API")
	c.SubThemesAIAPI = os.Getenv("APP_SUB_THEMES_AI_API")
	c.NetSentimentInterpretationAPI = os.Getenv("APP_net_sentiment_interpretation_api")
	c.ChatsTopKeywords = os.Getenv("APP_CHATS_TOPKEYWORDS")
	c.CDPProfileAPI = os.Getenv("APP_CDP_PROFILE_API")
	c.OverallSentimentURL = os.Getenv("APP_OVERALL_SENTIMENT_URL")
	c.AIAPIsTimeout = time.Duration(parseInt64(os.Getenv("APP_AI_APIS_TIMEOUT"), 60)) * time.Second
	c.BackendAPIURL = os.Getenv("APP_BACKEND_API_URL")
	c.LiveServiceURL = os.Getenv("APP_LIVE_SERVICE_URL")

	// Behavior
	c.EventTimeout = time.Duration(parseInt64(os.Getenv("APP_GO_DISPATCHER_EVENT_TIMEOUT"), 3600)) * time.Second
	c.MaxTries = int(parseInt64(os.Getenv("APP_SERVICE_MAX_TRIES"), 0))
	c.CrashSleep = time.Duration(parseInt64(os.Getenv("APP_SERVICE_CRASH_SLEEP"), 0)) * time.Second
	c.ForceDisableInactivity = parseBool(os.Getenv("APP_FORCE_DISABLE_INACTIVITY_THREAD"))
	c.ServerEnvironment = os.Getenv("APP_SERVER_ENVIRONMENT")
	c.TotalStaciPosts = parseInt64(os.Getenv("APP_TOTAL_NUMBER_OF_STACI_POSTS"), 0)

	// Whitelabel
	c.WhitelabelProjectNameEN = os.Getenv("APP_WHITELABEL_PROJECT_NAME_EN")
	c.WhitelabelAIProductNameEN = os.Getenv("APP_WHITELABEL_AI_PRODUCT_NAME_EN")
	c.WhitelabelProjectNameAR = os.Getenv("APP_WHITELABEL_PROJECT_NAME_AR")
	c.WhitelabelAIProductNameAR = os.Getenv("APP_WHITELABEL_AI_PRODUCT_NAME_AR")
	c.WhitelabelPoweredBy = os.Getenv("APP_WHITELABEL_POWERED_BY")
	c.WhitelabelLegalName = os.Getenv("APP_WHITELABEL_LEGAL_NAME")
	c.WhitelabelPoweredBySentence = os.Getenv("APP_WHITELABEL_POWERED_BY_SENTENCE")
	c.WhitelabelDisclaimerAR = os.Getenv("APP_WHITELABEL_DISCLAIMER_AR")
	c.WhitelabelContactEmail = os.Getenv("APP_WHITELABEL_CONTACT_EMAIL")
	c.WhitelabelAssumeResponsibility = parseBool(os.Getenv("APP_WHITELABEL_ASSUME_RESPONSIBILITY"))
	c.WhitelabelAIProductPPTLogo = os.Getenv("APP_WHITELABEL_AI_PRODUCT_PPT_LOGO")

	// Observability
	c.OTELServiceName = getEnv("OTEL_SERVICE_NAME", "publisher-engine")
	c.OTELTracesExporter = os.Getenv("OTEL_TRACES_EXPORTER")
	c.OTELMetricsExporter = os.Getenv("OTEL_METRICS_EXPORTER")
	c.OTELLogsExporter = os.Getenv("OTEL_LOGS_EXPORTER")
	c.OTELExporterOTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	c.LogFmt = parseBool(os.Getenv("APP_LOGFMT"))
	c.NoSlack = parseBool(os.Getenv("APP_NO_SLACK"))

	c.HTTPPort = getEnv("PORT", "8080")

	return c, c.validate()
}

func (c *Config) validate() error {
	if c.PublisherInputStream == "" {
		return fmt.Errorf("APP_PUBLISHER_INPUT_STREAM is required")
	}
	if c.PublisherConsumerGroup == "" {
		return fmt.Errorf("APP_PUBLISHER_CONSUMER_GROUP is required")
	}
	if c.PublisherConsumerName == "" {
		return fmt.Errorf("APP_PUBLISHER_CONSUMER_NAME is required")
	}
	if c.PublisherOutputStreamPrefix == "" {
		return fmt.Errorf("APP_PUBLISHER_OUTPUT_STREAM_PREFIX is required")
	}
	if c.PublisherDeadletterStream == "" {
		return fmt.Errorf("APP_PUBLISHER_DEADLETTER_STREAM is required")
	}
	return nil
}

// RedisAddr returns host:port for the main Redis broker.
func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// BackendRedisAddr returns host:port for backend Redis (buffer/event status).
func (c *Config) BackendRedisAddr() string {
	return c.BackendRedisHost + ":" + c.BackendRedisPort
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes"
}

func parseInt64(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}
