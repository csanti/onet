package log

import (
	"fmt"
	"log/syslog"
	"os"

	"github.com/daviddengcn/go-colortext"
)

// LoggerInfo is a structure that should be used when creating a logger.
// It contains parameters about how to log (with time, colors, ...) and
// embeds the Logger interface, which should define how the logger should log.
type LoggerInfo struct {
	// These are information-debugging levels that can be turned on or off.
	// Every logging greater than 'debugLvl' will be discarded . So you can
	// Log at different levels and easily turn on or off the amount of logging
	// generated by adjusting the 'debugLvl' variable.
	DebugLvl int
	// If 'showTime' is true, it will print the time for each line displayed
	// by the logger.
	ShowTime bool
	// If 'useColors' is true, logs will be colored (defaults to monochrome
	// output). It also controls padding, since colorful output is higly
	// correlated with humans who like their log lines padded.
	UseColors bool
	// If 'padding' is true, it will nicely pad the line that is written.
	Padding bool
}

// Logger is the interface that specifies how loggers
// will receive and display messages.
type Logger interface {
	Log(level int, msg string)
	Close()
	GetLoggerInfo() *LoggerInfo
}

const (
	// DefaultStdDebugLvl is the default debug level for the standard logger
	DefaultStdDebugLvl = 1
	// DefaultStdShowTime is the default value for 'showTime' for the standard logger
	DefaultStdShowTime = false
	// DefaultStdUseColors is the default value for 'useColors' for the standard logger
	DefaultStdUseColors = false
	// DefaultStdPadding is the default value for 'padding' for the standard logger
	DefaultStdPadding = true
)

var (
	// concurrent access is protected by debugMut
	loggers        = make(map[int]Logger)
	loggersCounter int
)

// RegisterLogger will register a callback that will receive a copy of every
// message, fully formatted. It returns the key assigned to the logger (used
// to unregister the logger).
func RegisterLogger(l Logger) int {
	debugMut.Lock()
	defer debugMut.Unlock()
	key := loggersCounter
	loggers[key] = l
	loggersCounter++
	return key
}

// UnregisterLogger takes the key it was assigned and returned by
// 'RegisterLogger', closes the corresponding Logger and removes it from the
// loggers.
func UnregisterLogger(key int) {
	debugMut.Lock()
	defer debugMut.Unlock()
	if l, ok := loggers[key]; ok {
		l.Close()
		delete(loggers, key)
	}
}

type fileLogger struct {
	lInfo *LoggerInfo
	file  *os.File
}

func (fl *fileLogger) Log(level int, msg string) {
	if _, err := fl.file.WriteString(msg); err != nil {
		panic(err)
	}
}

func (fl *fileLogger) Close() {
	fl.file.Close()
}

func (fl *fileLogger) GetLoggerInfo() *LoggerInfo {
	return fl.lInfo
}

// NewFileLogger creates a logger that writes into the file with
// the given path and is using the given LoggerInfo.
// It returns the logger.
func NewFileLogger(lInfo *LoggerInfo, path string) (Logger, error) {
	// Override file if it already exists.
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &fileLogger{
		lInfo: lInfo,
		file:  file,
	}, nil
}

type syslogLogger struct {
	lInfo  *LoggerInfo
	writer *syslog.Writer
}

func (sl *syslogLogger) Log(level int, msg string) {
	_, err := sl.writer.Write([]byte(msg))
	if err != nil {
		panic(err)
	}
}

func (sl *syslogLogger) Close() {
	sl.writer.Close()
}

func (sl *syslogLogger) GetLoggerInfo() *LoggerInfo {
	return sl.lInfo
}

// NewSyslogLogger creates a logger that writes into syslog with
// the given priority and tag, and is using the given LoggerInfo (without the
// Logger).
// It returns the logger.
func NewSyslogLogger(lInfo *LoggerInfo, priority syslog.Priority, tag string) (Logger, error) {
	writer, err := syslog.New(priority, tag)
	if err != nil {
		return nil, err
	}
	return &syslogLogger{
		lInfo:  lInfo,
		writer: writer,
	}, nil
}

type stdLogger struct {
	lInfo *LoggerInfo
}

func (sl *stdLogger) Log(lvl int, msg string) {
	if sl.lInfo.UseColors {
		bright := lvl < 0
		lvlAbs := lvl
		if bright {
			lvlAbs *= -1
		}

		switch lvl {
		case lvlPrint:
			ct.Foreground(ct.White, true)
		case lvlInfo:
			ct.Foreground(ct.White, true)
		case lvlWarning:
			ct.Foreground(ct.Green, true)
		case lvlError:
			ct.Foreground(ct.Red, false)
		case lvlFatal:
			ct.Foreground(ct.Red, true)
		case lvlPanic:
			ct.Foreground(ct.Red, true)
		default:
			if lvl != 0 {
				if lvlAbs <= 5 {
					colors := []ct.Color{ct.Yellow, ct.Cyan, ct.Green, ct.Blue, ct.Cyan}
					ct.Foreground(colors[lvlAbs-1], bright)
				}
			}
		}
	}

	if lvl < lvlInfo {
		fmt.Fprint(stdErr, msg)
	} else {
		fmt.Fprint(stdOut, msg)
	}

	if sl.lInfo.UseColors {
		ct.ResetColor()
	}
}

func (sl *stdLogger) Close() {}

func (sl *stdLogger) GetLoggerInfo() *LoggerInfo {
	return sl.lInfo
}

// Not public + not taking a LoggerInfo as argument because we don't want
// multiple stdLoggers.
func newStdLogger() (Logger, error) {
	lInfo := &LoggerInfo{
		DebugLvl:  DefaultStdDebugLvl,
		UseColors: DefaultStdUseColors,
		ShowTime:  DefaultStdShowTime,
		Padding:   DefaultStdPadding,
	}
	return &stdLogger{lInfo: lInfo}, nil
}
