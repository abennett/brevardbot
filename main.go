package main

import (
	"errors"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	tele "gopkg.in/telebot.v3"
)

const (
	envToken  = "TELEGRAM_TOKEN"
	envDebug  = "DEBUG"
	envPort   = "PORT"
	envBotURL = "BOT_URL"
)

var (
	errNoPortProvided = errors.New("provide a port via PORT envvar")
	errNoURLProvided  = errors.New("provide a url")
)

func setupWebhook() (*tele.Webhook, error) {
	port, ok := os.LookupEnv(envPort)
	if !ok {
		return nil, errNoPortProvided
	}
	url, ok := os.LookupEnv(envBotURL)
	if !ok {
		return nil, errNoURLProvided
	}
	wh := &tele.Webhook{
		Listen: ":" + port,
		// 1 - 100
		MaxConnections: 50,
		DropUpdates:    true,
		Endpoint: &tele.WebhookEndpoint{
			PublicURL: url,
		},
	}
	return wh, nil
}

func setupLogger() *zap.Logger {
	logEncoding := zap.NewProductionEncoderConfig()
	logEncoding.EncodeTime = zapcore.RFC3339TimeEncoder
	logConfig := zap.NewProductionConfig()
	logConfig.DisableStacktrace = true
	logConfig.DisableCaller = true
	logConfig.EncoderConfig = logEncoding
	if _, ok := os.LookupEnv(envDebug); ok {
		logConfig.Level.SetLevel(zap.DebugLevel)
	}
	logger, err := logConfig.Build()
	if err != nil {
		panic("logger failed")
	}
	return logger
}

func main() {
	logger := setupLogger().Named("brevardbot")
	wh, err := setupWebhook()
	if err != nil {
		logger.Fatal("failed to create webhook poller", zap.Error(err))
	}
	settings := tele.Settings{
		Token:  os.Getenv(envToken),
		Poller: wh,
	}
	cd := newCountdownBox(logger)
	b, err := tele.NewBot(settings)
	if err != nil {
		logger.Fatal("failed to create bot", zap.Error(err))
	}
	b.Handle("/countdown", cd.countdown)
	b.Handle("/cancel", cd.cancel)
	logger.Info("Starting BrevardBot")
	b.Start()
}
