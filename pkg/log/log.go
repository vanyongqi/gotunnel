package log

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed i18n/*.toml
var i18nFiles embed.FS

// Level represents the log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

var (
	currentLevel Level
	currentLang  language.Tag
	mu           sync.RWMutex
	bundle       *i18n.Bundle
	localizer    *i18n.Localizer
)

// Init initializes the logger with level and language
func Init(level Level, lang language.Tag) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
	currentLang = lang

	// Initialize i18n bundle
	bundle = i18n.NewBundle(lang)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Load translation files
	loadTranslations()

	// Create localizer
	localizer = i18n.NewLocalizer(bundle, lang.String())
}

// SetLevel sets the log level
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

// SetLanguage sets the log language
func SetLanguage(lang language.Tag) {
	mu.Lock()
	defer mu.Unlock()
	currentLang = lang
	if bundle != nil {
		localizer = i18n.NewLocalizer(bundle, lang.String())
	}
}

// GetLevel returns the current log level
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// GetLanguage returns the current log language
func GetLanguage() language.Tag {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// loadTranslations loads translation files from embedded filesystem
func loadTranslations() {
	// Load Chinese translations
	if data, err := i18nFiles.ReadFile("i18n/active.zh.toml"); err == nil {
		bundle.MustParseMessageFileBytes(data, "active.zh.toml")
	}

	// Load English translations
	if data, err := i18nFiles.ReadFile("i18n/active.en.toml"); err == nil {
		bundle.MustParseMessageFileBytes(data, "active.en.toml")
	}
}

// shouldLog checks if the log level should be logged
func shouldLog(level Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level >= currentLevel
}

// log prints a log message with level, module, and message
func log(level Level, module, key string, data map[string]interface{}) {
	if !shouldLog(level) {
		return
	}

	mu.RLock()
	levelName := levelNames[level]
	loc := localizer
	mu.RUnlock()

	if loc == nil {
		// Fallback if not initialized
		output := fmt.Sprintf("[%s][%s] %s\n", levelName, module, key)
		if level >= LevelError {
			os.Stderr.WriteString(output)
		} else {
			os.Stdout.WriteString(output)
		}
		return
	}

	// Get translated message
	msg, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: data,
	})
	if err != nil {
		// Fallback to key if translation not found
		msg = key
	}

	// Format: [LEVEL][MODULE] message
	output := fmt.Sprintf("[%s][%s] %s\n", levelName, module, msg)

	if level >= LevelError {
		os.Stderr.WriteString(output)
	} else {
		os.Stdout.WriteString(output)
	}
}

// Debug logs a debug message
func Debug(module, key string, data map[string]interface{}) {
	log(LevelDebug, module, key, data)
}

// Info logs an info message
func Info(module, key string, data map[string]interface{}) {
	log(LevelInfo, module, key, data)
}

// Warn logs a warning message
func Warn(module, key string, data map[string]interface{}) {
	log(LevelWarn, module, key, data)
}

// Error logs an error message
func Error(module, key string, data map[string]interface{}) {
	log(LevelError, module, key, data)
}

// Helper functions for common cases with variadic arguments
func Debugf(module, key string, args ...interface{}) {
	data := argsToMap(args)
	Debug(module, key, data)
}

func Infof(module, key string, args ...interface{}) {
	data := argsToMap(args)
	Info(module, key, data)
}

func Warnf(module, key string, args ...interface{}) {
	data := argsToMap(args)
	Warn(module, key, data)
}

func Errorf(module, key string, args ...interface{}) {
	data := argsToMap(args)
	Error(module, key, data)
}

// argsToMap converts variadic arguments to a map for template data
// This is a simple helper - in practice you might want more sophisticated mapping
func argsToMap(args []interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	for i, arg := range args {
		// Use common field names based on position
		switch i {
		case 0:
			if port, ok := arg.(int); ok {
				data["Port"] = port
			} else if str, ok := arg.(string); ok {
				data["Addr"] = str
				data["Name"] = str
				data["Target"] = str
				data["Reason"] = str
			} else if err, ok := arg.(error); ok {
				data["Error"] = err.Error()
			} else {
				data["Arg1"] = arg
			}
		case 1:
			if port, ok := arg.(int); ok {
				if localPort, exists := data["Port"]; exists {
					data["LocalPort"] = localPort
					data["RemotePort"] = port
				} else {
					data["RemotePort"] = port
				}
			} else if str, ok := arg.(string); ok {
				data["Token"] = str
			} else {
				data["Arg2"] = arg
			}
		default:
			data[fmt.Sprintf("Arg%d", i+1)] = arg
		}
	}
	return data
}

// ParseLevel parses a string to log level
func ParseLevel(levelStr string) Level {
	levelStr = strings.ToLower(levelStr)
	switch levelStr {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// ParseLanguage parses a string to language tag
func ParseLanguage(langStr string) language.Tag {
	langStr = strings.ToLower(langStr)
	switch langStr {
	case "zh", "zh-cn", "chinese":
		return language.Chinese
	case "en", "english":
		return language.English
	default:
		return language.Chinese
	}
}
