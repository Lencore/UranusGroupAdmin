package telegram

import (
	"app/dto"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v4"
)

// ModeratedGroup представляет группу, которая модерируется ботом
type ModeratedGroup struct {
	ChatID            int64
	CloseTime         string // Формат: "HH:MM"
	OpenTime          string // Формат: "HH:MM"
	WhitelistedLinks  []string
	WhitelistedUsers  []int64
	EveningMessage    string
	MorningMessage    string
	ModerateLinks     bool
	ModerateScheduled bool
}

// Добавляем модели для сохранения в базе данных
func (t *Telegram) setupModeration() {
	// Регистрируем команды модерации
	t.bot.Handle("/moderate", t.cmdModerate)
	t.bot.Handle("/open", t.cmdSetOpenTime)
	t.bot.Handle("/close", t.cmdSetCloseTime)
	t.bot.Handle("/whitelist", t.cmdWhitelist)
	t.bot.Handle("/evening_message", t.cmdSetEveningMessage)
	t.bot.Handle("/morning_message", t.cmdSetMorningMessage)

	// Обработчик для всех сообщений (проверка ссылок)
	t.bot.Handle(tele.OnText, t.moderateLinks)

	// Запускаем планировщик для проверки времени открытия/закрытия чатов
	go t.scheduleModeration()
}

// cmdModerate обрабатывает команду /moderate
func (t *Telegram) cmdModerate(ctx tele.Context) error {
	// Проверяем, что команда пришла из группы
	if !ctx.Chat().IsGroup() {
		return ctx.Reply("Эта команда доступна только в группах")
	}

	// Проверяем, что отправитель - администратор
	sender := ctx.Sender()
	admins, err := t.bot.AdminsOf(ctx.Chat())
	if err != nil {
		zap.L().Error("Не удалось получить список администраторов", zap.Error(err))
		return ctx.Reply("Ошибка при проверке прав администратора")
	}

	isAdmin := false
	for _, admin := range admins {
		if admin.User.ID == sender.ID {
			isAdmin = true
			break
		}
	}

	if !isAdmin && sender.ID != dto.GlobalAdminID {
		return ctx.Reply("Только администраторы могут использовать эту команду")
	}

	// Создаем или обновляем запись о модерируемой группе
	group := &ModeratedGroup{
		ChatID:            ctx.Chat().ID,
		CloseTime:         "22:00",
		OpenTime:          "09:00",
		WhitelistedLinks:  []string{},
		WhitelistedUsers:  []int64{},
		EveningMessage:    "Чат закрыт до утра. Доброй ночи! 🌙",
		MorningMessage:    "Доброе утро! Чат открыт. 🌞",
		ModerateLinks:     true,
		ModerateScheduled: true,
	}

	// Здесь нужно сохранить группу в базу данных
	err = t.saveModeratedGroup(group)
	if err != nil {
		zap.L().Error("Не удалось сохранить группу", zap.Error(err))
		return ctx.Reply("Ошибка при настройке модерации")
	}

	return ctx.Reply("Режим модерации включен для этой группы. По умолчанию:\n" +
		"- Чат закрывается в 22:00\n" +
		"- Чат открывается в 09:00\n" +
		"- Модерация ссылок включена\n\n" +
		"Настройки можно изменить командами:\n" +
		"/close ЧЧ:ММ - время закрытия чата\n" +
		"/open ЧЧ:ММ - время открытия чата\n" +
		"/whitelist слово - добавить слово/ссылку в белый список")
}

// cmdSetOpenTime устанавливает время открытия чата
func (t *Telegram) cmdSetOpenTime(ctx tele.Context) error {
	if !ctx.Chat().IsGroup() {
		return ctx.Reply("Эта команда доступна только в группах")
	}

	// Проверяем права администратора
	if !t.isAdmin(ctx.Chat(), ctx.Sender()) {
		return ctx.Reply("Только администраторы могут использовать эту команду")
	}

	// Получаем аргументы команды
	args := ctx.Args()
	if len(args) != 1 {
		return ctx.Reply("Пожалуйста, укажите время в формате ЧЧ:ММ, например: /open 09:00")
	}

	// Проверяем формат времени
	timeStr := args[0]
	if !t.isValidTimeFormat(timeStr) {
		return ctx.Reply("Некорректный формат времени. Используйте формат ЧЧ:ММ, например: 09:00")
	}

	// Получаем текущие настройки группы
	group, err := t.getModeratedGroup(ctx.Chat().ID)
	if err != nil {
		return ctx.Reply("Эта группа не настроена для модерации. Используйте сначала команду /moderate")
	}

	// Обновляем время открытия
	group.OpenTime = timeStr
	err = t.saveModeratedGroup(group)
	if err != nil {
		zap.L().Error("Не удалось обновить настройки группы", zap.Error(err))
		return ctx.Reply("Ошибка при обновлении настроек")
	}

	return ctx.Reply(fmt.Sprintf("Время открытия чата установлено на %s", timeStr))
}

// cmdSetCloseTime устанавливает время закрытия чата
func (t *Telegram) cmdSetCloseTime(ctx tele.Context) error {
	if !ctx.Chat().IsGroup() {
		return ctx.Reply("Эта команда доступна только в группах")
	}

	// Проверяем права администратора
	if !t.isAdmin(ctx.Chat(), ctx.Sender()) {
		return ctx.Reply("Только администраторы могут использовать эту команду")
	}

	// Получаем аргументы команды
	args := ctx.Args()
	if len(args) != 1 {
		return ctx.Reply("Пожалуйста, укажите время в формате ЧЧ:ММ, например: /close 22:00")
	}

	// Проверяем формат времени
	timeStr := args[0]
	if !t.isValidTimeFormat(timeStr) {
		return ctx.Reply("Некорректный формат времени. Используйте формат ЧЧ:ММ, например: 22:00")
	}

	// Получаем текущие настройки группы
	group, err := t.getModeratedGroup(ctx.Chat().ID)
	if err != nil {
		return ctx.Reply("Эта группа не настроена для модерации. Используйте сначала команду /moderate")
	}

	// Обновляем время закрытия
	group.CloseTime = timeStr
	err = t.saveModeratedGroup(group)
	if err != nil {
		zap.L().Error("Не удалось обновить настройки группы", zap.Error(err))
		return ctx.Reply("Ошибка при обновлении настроек")
	}

	return ctx.Reply(fmt.Sprintf("Время закрытия чата установлено на %s", timeStr))
}

// cmdWhitelist обрабатывает команду /whitelist для добавления слов/ссылок в белый список
func (t *Telegram) cmdWhitelist(ctx tele.Context) error {
	// Эта команда доступна только в личке и только для глобального админа
	if !ctx.Chat().IsPrivate() {
		return ctx.Reply("Эта команда доступна только в личной переписке с ботом")
	}

	if ctx.Sender().ID != dto.GlobalAdminID {
		return ctx.Reply("Эта команда доступна только для главного администратора")
	}

	args := ctx.Args()
	if len(args) < 1 {
		return ctx.Reply("Пожалуйста, укажите слово или регулярное выражение для добавления в белый список")
	}

	// Добавляем слово в глобальный белый список
	whitelist := strings.Join(args, " ")
	err := t.addToGlobalWhitelist(whitelist)
	if err != nil {
		zap.L().Error("Не удалось добавить в белый список", zap.Error(err))
		return ctx.Reply("Ошибка при добавлении в белый список")
	}

	return ctx.Reply(fmt.Sprintf("'%s' добавлено в глобальный белый список", whitelist))
}

// cmdSetEveningMessage устанавливает сообщение, которое отправляется при закрытии чата
func (t *Telegram) cmdSetEveningMessage(ctx tele.Context) error {
	if !ctx.Chat().IsGroup() {
		return ctx.Reply("Эта команда доступна только в группах")
	}

	// Проверяем права администратора
	if !t.isAdmin(ctx.Chat(), ctx.Sender()) {
		return ctx.Reply("Только администраторы могут использовать эту команду")
	}

	// Получаем текст сообщения
	message := strings.Join(ctx.Args(), " ")
	if message == "" {
		return ctx.Reply("Пожалуйста, укажите текст вечернего сообщения")
	}

	// Получаем текущие настройки группы
	group, err := t.getModeratedGroup(ctx.Chat().ID)
	if err != nil {
		return ctx.Reply("Эта группа не настроена для модерации. Используйте сначала команду /moderate")
	}

	// Обновляем сообщение
	group.EveningMessage = message
	err = t.saveModeratedGroup(group)
	if err != nil {
		zap.L().Error("Не удалось обновить настройки группы", zap.Error(err))
		return ctx.Reply("Ошибка при обновлении настроек")
	}

	return ctx.Reply("Вечернее сообщение обновлено")
}

// cmdSetMorningMessage устанавливает сообщение, которое отправляется при открытии чата
func (t *Telegram) cmdSetMorningMessage(ctx tele.Context) error {
	if !ctx.Chat().IsGroup() {
		return ctx.Reply("Эта команда доступна только в группах")
	}

	// Проверяем права администратора
	if !t.isAdmin(ctx.Chat(), ctx.Sender()) {
		return ctx.Reply("Только администраторы могут использовать эту команду")
	}

	// Получаем текст сообщения
	message := strings.Join(ctx.Args(), " ")
	if message == "" {
		return ctx.Reply("Пожалуйста, укажите текст утреннего сообщения")
	}

	// Получаем текущие настройки группы
	group, err := t.getModeratedGroup(ctx.Chat().ID)
	if err != nil {
		return ctx.Reply("Эта группа не настроена для модерации. Используйте сначала команду /moderate")
	}

	// Обновляем сообщение
	group.MorningMessage = message
	err = t.saveModeratedGroup(group)
	if err != nil {
		zap.L().Error("Не удалось обновить настройки группы", zap.Error(err))
		return ctx.Reply("Ошибка при обновлении настроек")
	}

	return ctx.Reply("Утреннее сообщение обновлено")
}

// moderateLinks обрабатывает все текстовые сообщения для модерации ссылок
func (t *Telegram) moderateLinks(ctx tele.Context) error {
	// Обрабатываем только сообщения в группах
	if !ctx.Chat().IsGroup() {
		return nil
	}

	// Проверяем, включена ли модерация в этой группе
	group, err := t.getModeratedGroup(ctx.Chat().ID)
	if err != nil || !group.ModerateLinks {
		return nil
	}

	// Если отправитель - администратор, не модерируем
	if t.isAdmin(ctx.Chat(), ctx.Sender()) {
		return nil
	}

	// Если пользователь в белом списке, не модерируем
	if t.isUserWhitelisted(ctx.Sender().ID, group) {
		return nil
	}

	// Проверяем наличие ссылок в сообщении
	text := ctx.Text()
	if t.containsBlockedLink(text, group) {
		// Удаляем сообщение
		err := ctx.Delete()
		if err != nil {
			zap.L().Error("Не удалось удалить сообщение", zap.Error(err))
			return nil
		}

		// Уведомляем пользователя (в личку)
		t.bot.Send(&tele.User{ID: ctx.Sender().ID}, "Ваше сообщение было удалено, так как оно содержит ссылки. Если вы считаете, что это ошибка, обратитесь к администраторам группы.")

		return nil
	}

	return nil
}

// scheduleModeration запускает планировщик для проверки времени открытия/закрытия чатов
func (t *Telegram) scheduleModeration() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.checkGroupsSchedule()
		}
	}
}

// checkGroupsSchedule проверяет расписание для всех модерируемых групп
func (t *Telegram) checkGroupsSchedule() {
	// Получаем текущее время
	now := time.Now()
	currentTimeStr := now.Format("15:04")

	// Получаем все модерируемые группы
	groups, err := t.getAllModeratedGroups()
	if err != nil {
		zap.L().Error("Не удалось получить список модерируемых групп", zap.Error(err))
		return
	}

	for _, group := range groups {
		if !group.ModerateScheduled {
			continue
		}

		// Проверяем, совпадает ли текущее время с временем открытия/закрытия
		if currentTimeStr == group.OpenTime {
			t.openChat(group)
		} else if currentTimeStr == group.CloseTime {
			t.closeChat(group)
		}
	}
}

// openChat открывает чат для отправки сообщений
func (t *Telegram) openChat(group *ModeratedGroup) {
	chat := &tele.Chat{ID: group.ChatID}

	// Устанавливаем разрешения для отправки сообщений
	permissions := tele.ChatPermissions{
		CanSendMessages:       true,
		CanSendMediaMessages:  true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
		CanChangeInfo:         false,
		CanInviteUsers:        true,
		CanPinMessages:        false,
	}

	err := t.bot.SetGroupPermissions(chat, permissions)
	if err != nil {
		zap.L().Error("Не удалось открыть чат", zap.Error(err), zap.Int64("chat_id", group.ChatID))
		return
	}

	// Отправляем утреннее сообщение
	_, err = t.bot.Send(chat, group.MorningMessage)
	if err != nil {
		zap.L().Error("Не удалось отправить утреннее сообщение", zap.Error(err), zap.Int64("chat_id", group.ChatID))
	}

	zap.L().Info("Чат открыт", zap.Int64("chat_id", group.ChatID))
}

// closeChat закрывает чат для отправки сообщений
func (t *Telegram) closeChat(group *ModeratedGroup) {
	chat := &tele.Chat{ID: group.ChatID}

	// Устанавливаем запрет на отправку сообщений
	permissions := tele.ChatPermissions{
		CanSendMessages:       false,
		CanSendMediaMessages:  false,
		CanSendPolls:          false,
		CanSendOtherMessages:  false,
		CanAddWebPagePreviews: false,
		CanChangeInfo:         false,
		CanInviteUsers:        false,
		CanPinMessages:        false,
	}

	err := t.bot.SetGroupPermissions(chat, permissions)
	if err != nil {
		zap.L().Error("Не удалось закрыть чат", zap.Error(err), zap.Int64("chat_id", group.ChatID))
		return
	}

	// Отправляем вечернее сообщение
	_, err = t.bot.Send(chat, group.EveningMessage)
	if err != nil {
		zap.L().Error("Не удалось отправить вечернее сообщение", zap.Error(err), zap.Int64("chat_id", group.ChatID))
	}

	zap.L().Info("Чат закрыт", zap.Int64("chat_id", group.ChatID))
}

// Вспомогательные функции

// isAdmin проверяет, является ли пользователь администратором чата
func (t *Telegram) isAdmin(chat *tele.Chat, user *tele.User) bool {
	if user.ID == dto.GlobalAdminID {
		return true
	}

	admins, err := t.bot.AdminsOf(chat)
	if err != nil {
		zap.L().Error("Не удалось получить список администраторов", zap.Error(err))
		return false
	}

	for _, admin := range admins {
		if admin.User.ID == user.ID {
			return true
		}
	}

	return false
}

// isValidTimeFormat проверяет корректность формата времени
func (t *Telegram) isValidTimeFormat(timeStr string) bool {
	re := regexp.MustCompile(`^([01]?[0-9]|2[0-3]):([0-5][0-9])$`)
	return re.MatchString(timeStr)
}

// containsBlockedLink проверяет, содержит ли текст заблокированные ссылки
func (t *Telegram) containsBlockedLink(text string, group *ModeratedGroup) bool {
	// Регулярные выражения для проверки различных типов ссылок
	urlRe := regexp.MustCompile(`https?://\S+`)
	tMeRe := regexp.MustCompile(`t\.me/\S+`)
	atMentionRe := regexp.MustCompile(`@\w+`)

	// Получаем глобальный белый список
	globalWhitelist, err := t.getGlobalWhitelist()
	if err != nil {
		zap.L().Error("Не удалось получить глобальный белый список", zap.Error(err))
		globalWhitelist = []string{}
	}

	// Объединяем глобальный и локальный белые списки
	whitelist := append(globalWhitelist, group.WhitelistedLinks...)

	// Находим все совпадения
	urls := urlRe.FindAllString(text, -1)
	tMeLinks := tMeRe.FindAllString(text, -1)
	mentions := atMentionRe.FindAllString(text, -1)

	// Объединяем все найденные ссылки
	allLinks := append(urls, tMeLinks...)
	allLinks = append(allLinks, mentions...)

	// Проверяем каждую ссылку
	for _, link := range allLinks {
		isWhitelisted := false
		for _, allowed := range whitelist {
			if strings.Contains(strings.ToLower(link), strings.ToLower(allowed)) {
				isWhitelisted = true
				break
			}
		}

		if !isWhitelisted {
			return true
		}
	}

	return false
}

// isUserWhitelisted проверяет, находится ли пользователь в белом списке
func (t *Telegram) isUserWhitelisted(userID int64, group *ModeratedGroup) bool {
	for _, id := range group.WhitelistedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// Методы работы с базой данных для модерации

// saveModeratedGroup сохраняет настройки модерируемой группы в базу данных
func (t *Telegram) saveModeratedGroup(group *ModeratedGroup) error {
	// Здесь должен быть код для сохранения в базу данных
	// В примере используем Redis для хранения данных

	key := fmt.Sprintf("moderated_group:%d", group.ChatID)

	// Сериализуем данные в формат для хранения
	whitelistedLinks := strings.Join(group.WhitelistedLinks, ",")

	var whitelistedUsersStr []string
	for _, id := range group.WhitelistedUsers {
		whitelistedUsersStr = append(whitelistedUsersStr, strconv.FormatInt(id, 10))
	}
	whitelistedUsers := strings.Join(whitelistedUsersStr, ",")

	// Сохраняем поля группы
	err := t.redis.Set(key+":close_time", group.CloseTime)
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":open_time", group.OpenTime)
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":whitelisted_links", whitelistedLinks)
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":whitelisted_users", whitelistedUsers)
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":evening_message", group.EveningMessage)
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":morning_message", group.MorningMessage)
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":moderate_links", strconv.FormatBool(group.ModerateLinks))
	if err != nil {
		return err
	}

	err = t.redis.Set(key+":moderate_scheduled", strconv.FormatBool(group.ModerateScheduled))
	if err != nil {
		return err
	}

	return nil
}

// getModeratedGroup получает настройки модерируемой группы из базы данных
func (t *Telegram) getModeratedGroup(chatID int64) (*ModeratedGroup, error) {
	key := fmt.Sprintf("moderated_group:%d", chatID)

	// Проверяем существование записи
	if !t.redis.Has(key + ":close_time") {
		return nil, fmt.Errorf("group not found")
	}

	closeTime, err := t.redis.GetString(key + ":close_time")
	if err != nil {
		return nil, err
	}

	openTime, err := t.redis.GetString(key + ":open_time")
	if err != nil {
		return nil, err
	}

	whitelistedLinksStr, err := t.redis.GetString(key + ":whitelisted_links")
	if err != nil {
		return nil, err
	}

	whitelistedUsersStr, err := t.redis.GetString(key + ":whitelisted_users")
	if err != nil {
		return nil, err
	}

	eveningMessage, err := t.redis.GetString(key + ":evening_message")
	if err != nil {
		return nil, err
	}

	morningMessage, err := t.redis.GetString(key + ":morning_message")
	if err != nil {
		return nil, err
	}

	moderateLinksStr, err := t.redis.GetString(key + ":moderate_links")
	if err != nil {
		return nil, err
	}

	moderateScheduledStr, err := t.redis.GetString(key + ":moderate_scheduled")
	if err != nil {
		return nil, err
	}

	// Парсим данные
	var whitelistedLinks []string
	if whitelistedLinksStr != "" {
		whitelistedLinks = strings.Split(whitelistedLinksStr, ",")
	}

	var whitelistedUsers []int64
	if whitelistedUsersStr != "" {
		for _, idStr := range strings.Split(whitelistedUsersStr, ",") {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				continue
			}
			whitelistedUsers = append(whitelistedUsers, id)
		}
	}

	moderateLinks, _ := strconv.ParseBool(moderateLinksStr)
	moderateScheduled, _ := strconv.ParseBool(moderateScheduledStr)

	return &ModeratedGroup{
		ChatID:            chatID,
		CloseTime:         closeTime,
		OpenTime:          openTime,
		WhitelistedLinks:  whitelistedLinks,
		WhitelistedUsers:  whitelistedUsers,
		EveningMessage:    eveningMessage,
		MorningMessage:    morningMessage,
		ModerateLinks:     moderateLinks,
		ModerateScheduled: moderateScheduled,
	}, nil
}

// getAllModeratedGroups получает все модерируемые группы
func (t *Telegram) getAllModeratedGroups() ([]*ModeratedGroup, error) {
	// В реальном приложении здесь должен быть запрос к базе данных
	// В примере просто сканируем все ключи в Redis

	// Это упрощенная реализация, в реальности нужно использовать паттерн сканирования Redis
	// или хранить список групп в отдельном ключе

	var groups []*ModeratedGroup

	// Заглушка для примера
	// В реальном приложении здесь будет запрос к Redis для получения всех ключей с префиксом "moderated_group:"

	return groups, nil
}

// addToGlobalWhitelist добавляет слово/регулярное выражение в глобальный белый список
func (t *Telegram) addToGlobalWhitelist(word string) error {
	// Получаем текущий белый список
	whitelist, err := t.getGlobalWhitelist()
	if err != nil {
		// Если ошибка, создаем новый список
		whitelist = []string{}
	}

	// Добавляем новое слово
	whitelist = append(whitelist, word)

	// Сохраняем обновленный список
	whitelistStr := strings.Join(whitelist, ",")
	return t.redis.Set("global:whitelist", whitelistStr)
}

// getGlobalWhitelist получает глобальный белый список
func (t *Telegram) getGlobalWhitelist() ([]string, error) {
	whitelistStr, err := t.redis.GetString("global:whitelist")
	if err != nil {
		return nil, err
	}

	if whitelistStr == "" {
		return []string{}, nil
	}

	return strings.Split(whitelistStr, ","), nil
}
