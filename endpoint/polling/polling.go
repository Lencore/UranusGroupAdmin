package polling

import (
	"app/util"
	"context"
	tele "gopkg.in/telebot.v4"

	"go.uber.org/zap"
)

type Endpoint struct {
	bot *tele.Bot
}

func New(bot *tele.Bot) (*Endpoint, error) {
	return &Endpoint{
		bot: bot,
	}, nil
}

func (w *Endpoint) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		zap.L().Info("Polling endpoint closed")
		w.bot.Stop()
	}()

	util.Restart(func() {
		zap.L().Info("init polling")
		w.bot.Start()
		zap.L().Info("Polling ended")
	}, "starting polling")

	return nil
}
