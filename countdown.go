package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	noMoreThan = 60 * time.Minute
	noLessThan = 1 * time.Minute

	oneMin    = time.Minute
	fiveMin   = 5 * time.Minute
	tenMin    = 10 * time.Minute
	twentyMin = 20 * time.Minute
)

var (
	errDurationTooLong  = errors.New("provided timespan is too long")
	errDurationTooShort = errors.New("provided timespan is too short")

	minuteRanges = [][]time.Duration{
		{twentyMin, tenMin},
		{tenMin, fiveMin},
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func smallID() string {
	id := rand.Intn(999)
	return fmt.Sprintf("%03d", id)
}

type countdowns struct {
	ctx    context.Context
	mu     sync.Mutex
	box    map[string]context.CancelFunc
	logger *zap.Logger
}

func newCountdownBox(logger *zap.Logger) *countdowns {
	return &countdowns{
		ctx:    context.Background(),
		mu:     sync.Mutex{},
		box:    make(map[string]context.CancelFunc),
		logger: logger.Named("countdowns"),
	}
}

func (cd *countdowns) put(id string, f context.CancelFunc) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	cd.box[id] = f
}

func (cd *countdowns) stop(id string) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	if f, ok := cd.box[id]; ok {
		f()
		delete(cd.box, id)
		return nil
	}
	return fmt.Errorf("countdown #%s does not exist", id)
}

func (cd *countdowns) countdown(c tele.Context) error {
	id := smallID()
	logger := cd.logger.With(zap.String("request_id", id))
	payload := c.Message().Payload
    if payload == "" {
        c.Send("Please provide a countdown duration. (ex. 9m)")
    }
	d, err := time.ParseDuration(c.Message().Payload)
	if err != nil {
		logger.Error("request not in duration format", zap.String("payload", payload))
		return err
	}
	logger.Info("starting a new countdown",
		zap.String("total_duration", payload),
	)
	ctx, cancel := context.WithCancel(cd.ctx)
	defer cancel()
	cd.put(id, cancel)
	defer cd.stop(id)
	ch, err := minuteTimer(logger, ctx, d)
	if err != nil {
		logger.Error("failed to create timer", zap.Error(err))
		return err
	}
	if err = c.Send(fmt.Sprintf("Starting countdown from %s. #%s", payload, id)); err != nil {
		logger.Error("failed sending message", zap.Error(err))
		return err
	}
	<-ch //skip first
	for notify := range ch {
		logger.Debug("emitting", zap.String("duration", notify))
		if err = c.Send(notify); err != nil {
			logger.Error("failed sending message", zap.Error(err))
			return err
		}
	}
	logger.Info("finishing request")
	return nil
}

func (cd *countdowns) cancel(c tele.Context) error {
	payload := c.Message().Payload
	if err := cd.stop(payload); err != nil {
		if err = c.Send(err.Error()); err != nil {
			cd.logger.Error("failed to send message", zap.Error(err))
		}
		return err
	}
	if err := c.Send(fmt.Sprintf("countdown #%s canceled", payload)); err != nil {
		cd.logger.Error("failed to send message", zap.Error(err))
	}
	cd.logger.Info("countdown canceled", zap.String("countdown_id", payload))
	return nil
}

func formatMinutes(d time.Duration) string {
	return fmt.Sprintf("T-%dm", int(d.Minutes()))
}

func waitFor(d time.Duration) time.Duration {
	for _, mins := range minuteRanges {
		if d <= mins[0] {
			continue
		}
		if d-mins[1] < tenMin {
			return d - tenMin
		}
		return mins[1]
	}
	return oneMin
}

func minuteTimer(logger *zap.Logger, ctx context.Context, d time.Duration) (<-chan string, error) {
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
			// stop and close goroutine if canceled
			if ctx.Err() != nil {
				close(out)
				return
			}
			minutes := formatMinutes(totalMinutes)
			if totalMinutes <= 0 {
				break
			}
			out <- minutes
			wf := waitFor(totalMinutes)
			totalMinutes -= wf
			logger.Debug("sleeping", zap.Duration("sleep_duration", wf))
			time.Sleep(wf)
		}
		out <- "🏁"
		close(out)
	}()
	return out, nil
}
