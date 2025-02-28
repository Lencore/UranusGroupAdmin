package telegram

import (
	"app/gateway/database"
	"fmt"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
	"time"
)

func (t *Telegram) loadUser(ctx tele.Context) (*database.User, error) {
	user, err := t.db.GetUserByID(ctx.Sender().ID)
	if err != nil {
		user := &database.User{
			ID:        ctx.Sender().ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			FirstName: ctx.Sender().FirstName,
			LastName:  ctx.Sender().LastName,
			Username:  ctx.Sender().Username,
		}
		err := t.db.CreateUser(user)
		if err != nil {
			zap.L().Error("failed to create user", zap.Error(err))
			return nil, err
		}

		return user, nil
	}

	return user, err
}

func (t *Telegram) SomeCronJob() error {
	// например ты тут выбираешь юзеров и шлешь им что-то

	userIDs, err := t.db.GetAllUserIDs()
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		_, _ = t.bot.Send(&tele.User{ID: userID}, "Привет! Это сообщение из крон-задачи")
	}

	return nil
}

func (t *Telegram) cmdStart(ctx tele.Context) error {
	user, err := t.loadUser(ctx)
	if err != nil {
		zap.L().Error("failed to load user", zap.Error(err))
		_ = ctx.Send("Произошла ошибка при загрузке пользователя")
		return err
	}

	_ = ctx.Send(fmt.Sprintf("Привет, %s! 👋 Добро пожаловать!", user.FirstName))

	return nil
}

func (t *Telegram) cmdBan(ctx tele.Context) error {
	_ = ctx.Send("Эту команду только администраторы могут использовать")

	return nil
}

func (t *Telegram) cmdCountUsers(ctx tele.Context) error {
	// ты можешь юзать как готовые методы из database, так и юзать напрямую клиент базы (но я бы не советовал)
	// t.db.CountUsers() - готовый метод который надо сделать в gateway/database и вызывать его

	var totalUsers int64
	t.db.DB().Model(&database.User{}).Count(&totalUsers)

	_ = ctx.Send(fmt.Sprintf("Всего пользователей: %d", totalUsers))
	return nil
}
