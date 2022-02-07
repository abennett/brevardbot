package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	tele "gopkg.in/telebot.v3"
)

const (
	noMoreThan = 60 * time.Minute
	noLessThan = 1 * time.Minute

	fiveMin     = 5 * time.Minute
	minuteLimit = 10 * time.Minute

	envToken = "TELEGRAM_TOKEN"
	envDebug = "DEBUG"
)

var (
	errDurationTooLong  = errors.New("provided timespan is too long")
	errDurationTooShort = errors.New("provided timespan is too short")
)

func formatMinutes(d time.Duration) string {
	return fmt.Sprintf("T-%dm", int(d.Minutes()))
}

func waitFor(d time.Duration) time.Duration {
	if d > minuteLimit {
		if d-fiveMin < minuteLimit {
			return d - minuteLimit
		} else {
			return fiveMin
		}
	}
	return time.Minute
}

func minuteTimer(logger *zap.Logger, d time.Duration) (<-chan string, error) {
	if d > noMoreThan {
		return nil, errDurationTooLong
	}
	if d < noLessThan {
		return nil, errDurationTooShort
	}
	out := make(chan string, 1)
	go func() {
		totalMinutes := d.Truncate(time.Minute)
		for {
			minutes := formatMinutes(totalMinutes)
			out <- minutes
			if totalMinutes <= time.Minute {
				break
			}
			wf := waitFor(totalMinutes)
			totalMinutes -= wf
			logger.Debug("sleeping", zap.Duration("sleep_duration", wf))
			time.Sleep(wf)
		}
		close(out)
	}()
	return out, nil
}

func countdown(logger *zap.Logger) func(tele.Context) error {
	return func(c tele.Context) error {
		id := ulid.MustNew(ulid.Now(), rand.Reader)
		logger = logger.With(zap.String("request-id", id.String()))
		payload := c.Message().Payload
		d, err := time.ParseDuration(c.Message().Payload)
		if err != nil {
			logger.Error("request not in duration format", zap.String("payload", payload))
			return err
		}
		ch, err := minuteTimer(logger, d)
		if err != nil {
			logger.Error("failed to create timer", zap.Error(err))
			return err
		}
		logger.Info("start emitting", zap.String("total_duration", payload))
		for notify := range ch {
			logger.Debug("emitting", zap.String("duration", notify))
			err = c.Send(notify)
			if err != nil {
				logger.Error("failed sending message", zap.Error(err))
				return err
			}
		}
		c.Send("🏁")
		logger.Info("finishing request")
		return nil
	}
}

func setupLogger() *zap.Logger {
	logEncoding := zap.NewProductionEncoderConfig()
	logEncoding.EncodeTime = zapcore.RFC3339TimeEncoder
	logConfig := zap.NewProductionConfig()
	logConfig.DisableStacktrace = true
	logConfig.DisableCaller = true
	logConfig.EncoderConfig = logEncoding
	if _, ok := os.LookupEnv("DEBUG"); ok {
		logConfig.Level.SetLevel(zap.DebugLevel)
	}
	logger, err := logConfig.Build()
	if err != nil {
		panic("logger failed")
	}
	return logger
}

func main() {
	logger := setupLogger()
	settings := tele.Settings{
		Token:  os.Getenv(envToken),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(settings)
	if err != nil {
		logger.Fatal("failed to create bot", zap.Error(err))
	}
	b.Handle("/countdown", countdown(logger))
	logger.Info("Starting BrevardBot")
	b.Start()
}
