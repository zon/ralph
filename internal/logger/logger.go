package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	mu      sync.Mutex
	out     io.Writer
	json    bool
	level   Level
	verbose bool
}

type LoggerOption func(*Logger)

func WithJSONOutput() LoggerOption {
	return func(l *Logger) {
		l.json = true
	}
}

func WithOutput(w io.Writer) LoggerOption {
	return func(l *Logger) {
		l.out = w
	}
}

func NewLogger(options ...LoggerOption) *Logger {
	l := &Logger{
		out:   os.Stdout,
		json:  false,
		level: LevelInfo,
	}
	for _, opt := range options {
		opt(l)
	}
	return l
}

func (l *Logger) SetVerbose(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.verbose = enabled
}

func (l *Logger) logJSON(level, msg string) {
	entry := map[string]interface{}{
		"level": level,
		"msg":   msg,
	}
	data, _ := json.Marshal(entry)
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.out, string(data))
}

func (l *Logger) logPlain(level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "[%s] %s\n", level, msg)
}

func (l *Logger) log(level, msg string) {
	if l.json {
		l.logJSON(level, msg)
	} else {
		l.logPlain(level, msg)
	}
}

func (l *Logger) shouldLog(level Level) bool {
	return level >= l.level
}

func (l *Logger) Debug(msg string) {
	if !l.shouldLog(LevelDebug) {
		return
	}
	l.log("DEBUG", msg)
}

func (l *Logger) Debugf(format string, ctx ...interface{}) {
	if !l.shouldLog(LevelDebug) {
		return
	}
	l.log("DEBUG", fmt.Sprintf(format, ctx...))
}

func (l *Logger) Info(msg string) {
	if !l.shouldLog(LevelInfo) {
		return
	}
	l.log("INFO", msg)
}

func (l *Logger) Infof(format string, ctx ...interface{}) {
	if !l.shouldLog(LevelInfo) {
		return
	}
	l.log("INFO", fmt.Sprintf(format, ctx...))
}

func (l *Logger) Warn(msg string) {
	if !l.shouldLog(LevelWarn) {
		return
	}
	l.log("WARN", msg)
}

func (l *Logger) Warnf(format string, ctx ...interface{}) {
	if !l.shouldLog(LevelWarn) {
		return
	}
	l.log("WARN", fmt.Sprintf(format, ctx...))
}

func (l *Logger) Error(msg string) {
	if !l.shouldLog(LevelError) {
		return
	}
	l.log("ERROR", msg)
}

func (l *Logger) Errorf(format string, ctx ...interface{}) {
	if !l.shouldLog(LevelError) {
		return
	}
	l.log("ERROR", fmt.Sprintf(format, ctx...))
}

func (l *Logger) Success(msg string) {
	l.log("SUCCESS", msg)
}

func (l *Logger) Successf(format string, ctx ...interface{}) {
	l.log("SUCCESS", fmt.Sprintf(format, ctx...))
}

func (l *Logger) Verbose(msg string) {
	if !l.verbose {
		return
	}
	l.log("VERBOSE", msg)
}

func (l *Logger) Verbosef(format string, ctx ...interface{}) {
	if !l.verbose {
		return
	}
	l.log("VERBOSE", fmt.Sprintf(format, ctx...))
}

var (
	infoColor    = color.New(color.FgCyan)
	successColor = color.New(color.FgGreen)
	warningColor = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
	verboseColor = color.New(color.FgWhite, color.Faint)

	verboseEnabled = false
)

func SetVerbose(enabled bool) {
	verboseEnabled = enabled
}

func Info(msg string) {
	infoColor.Printf("[INFO] %s\n", msg)
}

func Infof(format string, ctx ...interface{}) {
	infoColor.Printf("[INFO] "+format+"\n", ctx...)
}

func Success(msg string) {
	successColor.Printf("✓ %s\n", msg)
}

func Successf(format string, ctx ...interface{}) {
	successColor.Printf("✓ "+format+"\n", ctx...)
}

func Warning(msg string) {
	warningColor.Printf("[WARNING] %s\n", msg)
}

func Warningf(format string, ctx ...interface{}) {
	warningColor.Printf("[WARNING] "+format+"\n", ctx...)
}

func Error(msg string) {
	errorColor.Printf("[ERROR] %s\n", msg)
}

func Errorf(format string, ctx ...interface{}) {
	errorColor.Printf("[ERROR] "+format+"\n", ctx...)
}

func Verbose(msg string) {
	if !verboseEnabled {
		return
	}
	verboseColor.Printf("[VERBOSE] %s\n", msg)
}

func Verbosef(format string, ctx ...interface{}) {
	if !verboseEnabled {
		return
	}
	verboseColor.Printf("[VERBOSE] "+format+"\n", ctx...)
}
