package docker

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var SupportedLogLevels map[string]struct{}

// Changing this also needs to change the logContext.ts file
var logLevels = []string{"error", "warn", "warning", "info", "debug", "trace", "severe", "critical", "fatal"}
var plainLevels = map[string]*regexp.Regexp{}
var bracketLevels = map[string]*regexp.Regexp{}

func init() {
	for _, level := range logLevels {
		plainLevels[level] = regexp.MustCompile("(?i)^" + level + "[^a-z]")
	}

	for _, level := range logLevels {
		bracketLevels[level] = regexp.MustCompile("(?i)\\[ ?" + level + " ?\\]")
	}

	SupportedLogLevels = make(map[string]struct{}, len(logLevels)+1)
	for _, level := range logLevels {
		SupportedLogLevels[level] = struct{}{}
	}
	SupportedLogLevels["unknown"] = struct{}{}
}

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		value = stripANSI(value)
		for _, level := range logLevels {
			// Look for the level at the beginning of the message
			if plainLevels[level].MatchString(value) {
				return level
			}

			// Look for the level in brackets
			if bracketLevels[level].MatchString(value) {
				return level
			}

			// Look for the level in the middle of the message that are uppercase
			if strings.Contains(value, " "+strings.ToUpper(level)+" ") {
				return level
			}

			// Look for levels with equal sign and quotes around them
			if strings.Contains(value, level+"=") {
				return level
			}
		}

		return "unknown"

	case *orderedmap.OrderedMap[string, any]:
		if value == nil {
			return "unknown"
		}

		if level, ok := value.Get("level"); ok {
			if level, ok := level.(string); ok {
				return normalizeLogLevel(level)
			}
		} else if severity, ok := value.Get("severity"); ok {
			if severity, ok := severity.(string); ok {
				return normalizeLogLevel(severity)
			}
		}

	case *orderedmap.OrderedMap[string, string]:
		if value == nil {
			return "unknown"
		}
		if level, ok := value.Get("level"); ok {
			return normalizeLogLevel(level)
		} else if severity, ok := value.Get("severity"); ok {
			return normalizeLogLevel(severity)
		}

	case map[string]interface{}:
		panic("not implemented")

	case map[string]string:
		panic("not implemented")

	default:
		log.Debug().Type("type", value).Msg("unknown logEvent type")
	}

	return "unknown"
}

func normalizeLogLevel(level string) string {
	level = stripANSI(level)
	level = strings.ToLower(level)
	if _, ok := SupportedLogLevels[level]; ok {
		return level
	}

	return "unknown"
}
