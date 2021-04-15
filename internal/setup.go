package internal

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"github.com/morikuni/failure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strconv"
	"strings"
)

func formatFrame(frame failure.Frame) string {
	return frame.Pkg() + "." + frame.Func() + ":" + strconv.Itoa(frame.Line())
}

func errorStackMarshaller(err error) interface{} {
	if cs, ok := failure.CallStackOf(err); ok {
		frames := cs.Frames()
		res := make([]string, 0, len(frames))
		for _, frame := range frames {
			res = append(res, formatFrame(frame))
		}
		return res
	}
	return err
}

func SetUpLogger(logLevel string, logFormat string) error {
	var writer io.Writer
	useConsoleWriter := false
	if logFormat == "auto" {
		if IsDev() {
			useConsoleWriter = true
		}
	} else if logFormat == "human" {
		useConsoleWriter = true
	} else if logFormat == "json" {
		useConsoleWriter = false
	} else {
		return fmt.Errorf("invalid log format: %s, expected: [auto, json, human]", logFormat)
	}
	if useConsoleWriter {
		writer = zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			if !isatty.IsTerminal(os.Stdout.Fd()) {
				w.NoColor = true
			}
		})
	} else {
		writer = os.Stdout
	}
	logger := zerolog.New(writer)
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = logger
	zerolog.ErrorStackMarshaler = errorStackMarshaller
	return nil
}
