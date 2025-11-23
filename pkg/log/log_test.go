package log

import (
	"os"
	"strings"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestLogLevel(t *testing.T) {
	Init(LevelInfo, language.Chinese)
	SetLevel(LevelInfo)

	// Capture output (both stdout and stderr)
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	Debug("test", "test.debug", map[string]interface{}{"Msg": "debug message"})
	Info("test", "test.info", map[string]interface{}{"Msg": "info message"})
	Warn("test", "test.warn", map[string]interface{}{"Msg": "warn message"})
	Error("test", "test.error", map[string]interface{}{"Msg": "error message"})

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read output from both
	bufOut := make([]byte, 1024)
	bufErr := make([]byte, 1024)
	nOut, _ := rOut.Read(bufOut)
	nErr, _ := rErr.Read(bufErr)
	output := string(bufOut[:nOut]) + string(bufErr[:nErr])

	// Debug should not appear
	if strings.Contains(output, "DEBUG") {
		t.Error("Debug message should not appear at Info level")
	}
	// Info, Warn, Error should appear
	if !strings.Contains(output, "INFO") {
		t.Error("Info message should appear")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("Warn message should appear")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Error message should appear")
	}
}

func TestTranslation(t *testing.T) {
	Init(LevelInfo, language.Chinese)

	SetLanguage(language.Chinese)
	msg, _ := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    "server.control_channel_listening",
		TemplateData: map[string]interface{}{"Addr": "test", "Token": "test"},
	})
	if msg == "" || !strings.Contains(msg, "控制通道") {
		t.Error("Chinese translation not working")
	}

	SetLanguage(language.English)
	msg, _ = localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    "server.control_channel_listening",
		TemplateData: map[string]interface{}{"Addr": "test", "Token": "test"},
	})
	if msg == "" || !strings.Contains(msg, "Control channel") {
		t.Error("English translation not working")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"unknown", LevelInfo},
	}

	for _, tt := range tests {
		result := ParseLevel(tt.input)
		if result != tt.expected {
			t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected language.Tag
	}{
		{"zh", language.Chinese},
		{"zh-cn", language.Chinese},
		{"chinese", language.Chinese},
		{"en", language.English},
		{"english", language.English},
		{"unknown", language.Chinese},
	}

	for _, tt := range tests {
		result := ParseLanguage(tt.input)
		if result != tt.expected {
			t.Errorf("ParseLanguage(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestGetLevel(t *testing.T) {
	Init(LevelDebug, language.Chinese)
	SetLevel(LevelWarn)
	if GetLevel() != LevelWarn {
		t.Errorf("GetLevel() = %v, want %v", GetLevel(), LevelWarn)
	}
}

func TestGetLanguage(t *testing.T) {
	Init(LevelInfo, language.English)
	SetLanguage(language.English)
	if GetLanguage() != language.English {
		t.Errorf("GetLanguage() = %v, want %v", GetLanguage(), language.English)
	}
}

func TestLogLevels(t *testing.T) {
	Init(LevelDebug, language.Chinese)
	
	// Capture output
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Test all levels at Debug level (should all appear)
	SetLevel(LevelDebug)
	Debug("test", "test.key", map[string]interface{}{"Msg": "debug"})
	Info("test", "test.key", map[string]interface{}{"Msg": "info"})
	Warn("test", "test.key", map[string]interface{}{"Msg": "warn"})
	Error("test", "test.key", map[string]interface{}{"Msg": "error"})

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	bufOut := make([]byte, 1024)
	bufErr := make([]byte, 1024)
	rOut.Read(bufOut)
	rErr.Read(bufErr)
	output := string(bufOut) + string(bufErr)

	if !strings.Contains(output, "DEBUG") {
		t.Error("Debug message should appear at Debug level")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("Info message should appear at Debug level")
	}
}

func TestHelperFunctions(t *testing.T) {
	Init(LevelInfo, language.Chinese)
	
	// Capture output
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Test helper functions
	Debugf("test", "test.key", "arg1", 123)
	Infof("test", "test.key", "arg1", 456)
	Warnf("test", "test.key", "arg1", 789)
	Errorf("test", "test.key", "arg1", "error")

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	bufOut := make([]byte, 1024)
	bufErr := make([]byte, 1024)
	rOut.Read(bufOut)
	rErr.Read(bufErr)
	output := string(bufOut) + string(bufErr)

	// Should contain INFO, WARN, ERROR (but not DEBUG at Info level)
	if strings.Contains(output, "DEBUG") {
		t.Error("Debug message should not appear at Info level")
	}
	if !strings.Contains(output, "INFO") || !strings.Contains(output, "WARN") || !strings.Contains(output, "ERROR") {
		t.Error("Helper functions should work correctly")
	}
}

func TestLogWithoutInit(t *testing.T) {
	// Test logging without initialization (should use fallback)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset to test fallback
	mu.Lock()
	localizer = nil
	mu.Unlock()

	Info("test", "test.key", map[string]interface{}{"Msg": "test"})

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "test.key") {
		t.Error("Fallback should output the key")
	}
}
