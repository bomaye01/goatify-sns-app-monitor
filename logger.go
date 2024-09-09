package main

import (
	"fmt"
	"log"
	"time"
)

const (
	reset  = "\033[0m"
	gray   = "\033[90m"
	red    = "\033[91m"
	green  = "\033[92m"
	yellow = "\033[93m"
	blue   = "\033[94m"
	pink   = "\033[95m"
	cyan   = "\033[96m"
	white  = "\033[97m"
)

type Logger struct {
	typeName string
}

func NewLogger(typeName string) *Logger {
	return &Logger{
		typeName: typeName,
	}
}

func (l *Logger) getPrefixCli() string {
	t := time.Now()

	return fmt.Sprintf("%s%02d.%02d.%d %02d:%02d:%02d.%02d%s %s[%s%s%s]%s", gray, t.Day(), t.Month(), t.Year()%100, t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1e7, reset, cyan, white, l.typeName, cyan, reset)
}

func (l *Logger) getPrefixFile() string {
	t := time.Now()

	return fmt.Sprintf("%02d.%02d.%d %02d:%02d:%02d.%02d [%s]", t.Day(), t.Month(), t.Year()%100, t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1e7, l.typeName)
}

func (l *Logger) Gray(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), gray, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) Red(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), red, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) Green(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), green, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) Yellow(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), yellow, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) Blue(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), blue, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) Pink(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), pink, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) Cyan(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), cyan, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}

func (l *Logger) White(format any) {
	log.Printf("%s %s%s%s\n", l.getPrefixCli(), white, format, reset)
	fileLogger.Printf("%s %s", l.getPrefixFile(), format)
}
