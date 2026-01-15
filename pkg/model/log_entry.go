package model

import (
	"time"
)

// LogLevel defines the severity of a log entry.
type LogLevel string

const (
	LogLevelFinest  LogLevel = "FINEST"
	LogLevelFiner   LogLevel = "FINER"
	LogLevelFine    LogLevel = "FINE"
	LogLevelConfig  LogLevel = "CONFIG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelSevere  LogLevel = "SEVERE"
)

// LogEntry requires these data points:
// log level, message, object (optional), logger name, time, sequence number, error, stacktrace.
// Common info: session id, client id, source.
// Extra info: ip address.
type LogEntry struct {
	// Fields
	Level      LogLevel               `json:"level,omitempty"` // Defaults to INFO
	Message    string                 `json:"message"`
	Object     map[string]interface{} `json:"object,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
	LoggerName string                 `json:"logger_name"`
	Time       time.Time              `json:"time"` // Defaults to Now
	Sequence   uint64                 `json:"sequence"`

	// Optional Error Details
	Error      string `json:"error,omitempty"`
	Stacktrace string `json:"stacktrace,omitempty"`

	// Common Context Fields (Session, Client, Source)
	SessionID string `json:"session_id"`
	ClientID  string `json:"client_id"`
	Source    string `json:"source"`

	// Enriched Fields
	ClientIP string `json:"client_ip,omitempty"`
}
