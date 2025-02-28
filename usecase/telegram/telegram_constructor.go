package telegram

import (
	"app/dto"
	"app/gateway/database"
	"app/gateway/redis"
	"context"
	tele "gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

type Telegram struct {
	config dto.Config
	redis  *redis.Redis
	db     *database.Database
	bot    *tele.Bot
}

func NewTelegram(
	config dto.Config,
	db *database.Database,
	redis *redis.Redis,
	bot *tele.Bot,
) (*Telegram, error) {
	return &Telegram{
		config: config,
		db:     db,
		redis:  redis,
		bot:    bot,
	}, nil
}

func (t *Telegram) Run(ctx context.Context) error {
	adminOnly := t.bot.Group()
	adminOnly.Use(middleware.Whitelist(dto.GlobalAdminID))
	adminOnly.Handle("/ban", t.cmdBan)

	t.bot.Handle("/start", t.cmdStart)
	t.bot.Handle("/stats", t.cmdCountUsers)

	return nil
}
