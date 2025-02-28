package cron

import (
	"go.uber.org/zap"
)

func (c *Cron) SimpleTask() {
	logger := zap.L().Named("Cron SimpleTask")

	logger.Info("started")
	// do something
	err := c.telegram.SomeCronJob()
	if err != nil {
		zap.L().Error("failed to send message to users", zap.Error(err))
		return
	}
	logger.Info("finished success")
}
