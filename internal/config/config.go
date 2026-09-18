package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"streamgo/internal/logger"
)

var log = logger.New("config")

// Config holds all configuration parameters for StreamGO.
type Config struct {
	Port          string
	Debug         bool
	APILogs       bool
	CorsOrigin    string
	MongoURI      string
	DatabaseName  string
	ApiID         int
	ApiHash       string
	BotToken      string
	SessionString string
	ChannelID     int64
	DumpChannelID int64
}

// Load reads configuration from .env file (if present) and environment variables.
func Load() *Config {
	// Try loading .env; ignore error if not found (e.g., in container/production)
	if err := godotenv.Load(); err != nil {
		log.Info("No .env file found, using system environment variables")
	} else {
		log.Info("Loaded configuration from .env")
	}

	debug := getEnvBool("DEBUG", false)
	logger.SetDebug(debug)

	mongoURI := strings.TrimSpace(getEnv("MONGO_URI", "mongodb://localhost:27017"))
	// Sanitize any accidental leading equals sign (e.g. MONGO_URI==...)
	mongoURI = strings.TrimPrefix(mongoURI, "=")

	return &Config{
		Port:          getEnv("PORT", "8000"),
		Debug:         debug,
		APILogs:       getEnvBool("API_LOGS", getEnvBool("HTTP_LOGS", false)),
		CorsOrigin:    getEnv("CORS_ORIGIN", "*"),
		MongoURI:      mongoURI,
		DatabaseName:  getEnv("DATABASE_NAME", "Stream"),
		ApiID:         getEnvInt("API_ID", 0),
		ApiHash:       strings.TrimSpace(os.Getenv("API_HASH")),
		BotToken:      strings.TrimSpace(os.Getenv("BOT_TOKEN")),
		SessionString: strings.TrimSpace(os.Getenv("SESSION_STRING")),
		ChannelID:     getEnvInt64("CHANNEL_ID", 0),
		DumpChannelID: getEnvInt64("DUMP_CHANNEL_ID", 0),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.ToLower(strings.TrimSpace(val))
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val, ok := os.LookupEnv(key); ok {
		if n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}
