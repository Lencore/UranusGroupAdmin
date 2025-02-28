package cron

import (
	"app/dto"
	"app/usecase/telegram"
	"context"
	"time"

	"github.com/go-co-op/gocron"
)

type Cron struct {
	cron     *gocron.Scheduler
	config   dto.Config
	telegram *telegram.Telegram
}

func NewCron(
	config dto.Config,
	telegram *telegram.Telegram,
) (*Cron, error) {
	timeLoc, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, err
	}

	s1 := gocron.NewScheduler(timeLoc)

	return &Cron{
		config:   config,
		cron:     s1,
		telegram: telegram,
	}, nil
}

func (c *Cron) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		c.cron.Stop()
	}()

	// каждые 5 минут
	_, err := c.cron.Every(5).Minute().Do(c.SimpleTask)
	if err != nil {
		return err
	}
	// или каждый день в 16:00
	//_, err = c.cron.Every(1).Day().At("16:00").Do(c.SimpleTask)

	c.cron.StartAsync()

	return nil
}
