package telegram

import (
	"app/dto"
	tele "gopkg.in/telebot.v4"
	"time"
)

func NewBot(config dto.Config) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:   config.Bot.Token,
		Poller:  &tele.LongPoller{Timeout: 10 * time.Second},
		Verbose: config.Bot.Debug,
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return b, nil
}
