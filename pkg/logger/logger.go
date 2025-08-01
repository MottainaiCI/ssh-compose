/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package logger

import (
	"fmt"
	"os"
	"regexp"

	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/kyokomi/emoji"
	"github.com/logrusorgru/aurora"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SshCLogger struct {
	Config *specs.SshComposeConfig
	Logger *zap.Logger
	Aurora aurora.Aurora
}

var defaultLogger *SshCLogger = nil

func NewSshCLogger(config *specs.SshComposeConfig) *SshCLogger {
	return &SshCLogger{
		Logger: nil,
		Aurora: aurora.NewAurora(config.GetLogging().Color),
		Config: config,
	}
}

func (l *SshCLogger) GetAurora() aurora.Aurora {
	return l.Aurora
}

func (l *SshCLogger) SetAsDefault() {
	defaultLogger = l
}

func GetDefaultLogger() *SshCLogger {
	return defaultLogger
}

func (l *SshCLogger) InitLogger2File() error {
	var err error

	// TODO: test permission for open logfile.
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{l.Config.GetLogging().Path}
	cfg.Level = level2AtomicLevel(l.Config.GetLogging().Level)
	cfg.ErrorOutputPaths = []string{}
	if l.Config.GetLogging().JsonFormat {
		cfg.Encoding = "json"
	} else {
		cfg.Encoding = "console"
	}
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l.Logger, err = cfg.Build()
	if err != nil {
		fmt.Fprint(os.Stderr, "Error on initialize file logger: "+err.Error()+"\n")
		return err
	}

	return nil
}

func level2Number(level string) int {
	switch level {
	case "error":
		return 0
	case "warning":
		return 1
	case "info":
		return 2
	default:
		return 3
	}
}

func (l *SshCLogger) log2File(level, msg string) {
	switch level {
	case "error":
		l.Logger.Error(msg)
	case "warning":
		l.Logger.Warn(msg)
	case "info":
		l.Logger.Info(msg)
	default:
		l.Logger.Debug(msg)
	}
}

func level2AtomicLevel(level string) zap.AtomicLevel {
	switch level {
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "warning":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	default:
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	}
}

func (l *SshCLogger) Msg(level string, withoutColor, ln bool, msg ...interface{}) {
	var message string
	var confLevel, msgLevel int

	if l.Config.GetGeneral().HasDebug() {
		confLevel = 3
	} else {
		confLevel = level2Number(l.Config.GetLogging().Level)
	}
	msgLevel = level2Number(level)
	if msgLevel > confLevel {
		return
	}

	for idx, m := range msg {
		if idx > 0 {
			message += " "
		}
		message += fmt.Sprintf("%v", m)
	}

	var levelMsg string

	if withoutColor || !l.Config.GetLogging().Color {
		levelMsg = message
	} else {
		switch level {
		case "warning":
			levelMsg = l.Aurora.Bold(l.Aurora.Yellow(":construction: " + message)).BgBlack().String()
		case "debug":
			levelMsg = l.Aurora.White(message).BgBlack().String()
		case "info":
			levelMsg = l.Aurora.Bold(l.Aurora.White(message)).BgBlack().String()
		case "error":
			levelMsg = l.Aurora.Bold(l.Aurora.Red(":bomb: " + message + ":fire:")).BgBlack().String()
		}
	}

	if l.Config.GetLogging().EnableEmoji {
		levelMsg = emoji.Sprint(levelMsg)
	} else {
		re := regexp.MustCompile(`[:][\w]+[:]`)
		levelMsg = re.ReplaceAllString(levelMsg, "")
	}

	if l.Logger != nil {
		l.log2File(level, message)
	}

	if ln {
		fmt.Println(levelMsg)
	} else {
		fmt.Print(levelMsg)
	}
}

func (l *SshCLogger) Warning(mess ...interface{}) {
	l.Msg("warning", false, true, mess...)
	//if l.FatalWarns {
	//	os.Exit(2)
	//}
}

func (l *SshCLogger) Debug(mess ...interface{}) {
	l.Msg("debug", false, true, mess...)
}

func (l *SshCLogger) DebugC(mess ...interface{}) {
	l.Msg("debug", true, true, mess...)
}

func (l *SshCLogger) Info(mess ...interface{}) {
	l.Msg("info", false, true, mess...)
}

func (l *SshCLogger) InfoC(mess ...interface{}) {
	l.Msg("info", true, true, mess...)
}

func (l *SshCLogger) Error(mess ...interface{}) {
	l.Msg("error", false, true, mess...)
}

func (l *SshCLogger) Fatal(mess ...interface{}) {
	l.Error(mess...)
	os.Exit(1)
}
