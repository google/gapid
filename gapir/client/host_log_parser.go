package client

import (
	"regexp"
	"strconv"
	"time"

	"github.com/google/gapid/core/log"
)

// "HH:MM:SS.FFF [VDIWEF] tag file:line : msg"
var hostLogMsgRegex = regexp.MustCompile(`\s*([0-9]*):([0-9]*):([0-9]*).([0-9]*)\s*([VDIWEF])\s*(\w*)\s*:\s*\[([a-zA-Z0-9_.]*):([0-9]*)\]\s(.*)`)

func parseHostLogMsg(s string) *log.Message {
	parts := hostLogMsgRegex.FindStringSubmatch(s)
	if parts == nil {
		return nil
	}
	hour, _ := strconv.Atoi(parts[1])
	minute, _ := strconv.Atoi(parts[2])
	second, _ := strconv.Atoi(parts[3])
	millisecs, _ := strconv.Atoi(parts[4])
	severity := parseHostLogPriority(parts[5][0])
	tag := parts[6]
	file := parts[7]
	line, _ := strconv.Atoi(parts[8])
	text := parts[9]
	now := time.Now()

	if tag == "gapir" {
		tag = ""
	}

	return &log.Message{
		Text:      text,
		Time:      time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, millisecs*1e6, time.Local),
		Severity:  severity,
		Tag:       tag,
		Process:   "gapir",
		Callstack: []*log.SourceLocation{&log.SourceLocation{File: file, Line: int32(line)}},
	}
}

func parseHostLogPriority(r byte) log.Severity {
	switch r {
	case 'V':
		return log.Verbose
	case 'D':
		return log.Debug
	case 'I':
		return log.Info
	case 'W':
		return log.Warning
	case 'E':
		return log.Error
	case 'F':
		return log.Fatal
	default:
		return log.Info
	}
}
