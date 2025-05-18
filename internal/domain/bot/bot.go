package bot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gitlab.com/gorib/pry"
)

const (
	CondominiumName = "Магнифика"

	StartShortcut       = "/start"
	ApplicationShortcut = "Заявка на въезд"
	EmergencyShortcut   = "Аварийно-диспетчерская служба"
	DispatcherShortcut  = "Заявка на въезд по телефону"
	GuardShortcut       = "Охрана"

	WelcomePhrase              = "Привет! Давай познакомимся! Пришли мне свой номер телефона, зарегистрированный в УК, в формате +7xxxxxxxxxx.\nНо имей в виду, что я запомню твой телефон. Его сможет увидеть только папа @tarandro (если захочет)."
	WelcomeAgainPhrase         = "Привет, рад видеть тебя снова! Если у тебя новый телефон, пришли мне его. Сейчас у меня записан: %s"
	ReadyForApplicationPhrase  = "Готово! Теперь можешь создавать заявки."
	OopsPhrase                 = "Ой, кажется я поломался! Позовите папу @tarandro!"
	NotFoundPhrase             = "УК не может найти тебя в списках. Свяжись с ними или укажи телефон, который зарегистрирован в УК."
	PhoneChangedPhrase         = "Что-то пошло не так и УК перестала тебя узнавать. Свяжись с УК или пришли мне свой новый номер телефона."
	ContactNotFoundPhrase      = "Что-то я потерял номер. Можешь попросить папу @tarandro помочь?"
	UnknownPersonPhrase        = "Извини, папа запрещает мне разговаривать с незнакомцами. Пришли мне свой номер телефона, зарегистрированный в УК."
	UnknownPhrase              = "Ой, что-то я не понял тебя. Что ты имеешь в виду?"
	UnknownTelegramErrorPhrase = "Telegram говорит что-то мне непонятное. Спроси папу @tarandro, может быть от знает."
	ApplicationFailedPhrase    = "Все было хорошо, но УК твою заявку не приняла. Не знаю, почему. Спроси папу @tarandro, он знает."
	ApplicationSentPhrase      = "Готово! Заявку отправил.\nВъезжать можно только с Магнитогорской улицы."
	WaitForPlatePhrase         = "Скажи, кого надо пропустить, и я передам дальше.\nМне нужен полный номер с регионом.\nНапример а000аа78."
)

type CustomerRepository interface {
	PhoneForCustomer(ctx context.Context, id int64) (string, error)
	SaveCustomer(ctx context.Context, id int64, phone string) error
}

type PhoneRepository interface {
	ValidatePhone(ctx context.Context, phone string) ([]string, error)
}

type ApplicationService interface {
	Application(ctx context.Context, phone, plate string, gates []string) error
}

func NewBotManagement(
	phones map[string]string,
	token string,
	repository CustomerRepository,
	validator PhoneRepository,
	application ApplicationService,
	logger pry.Logger,
) (*botManagement, error) {
	return &botManagement{
		logger:      logger,
		repository:  repository,
		validator:   validator,
		application: application,
		phones:      phones,
		token:       token,
	}, nil
}

type botManagement struct {
	logger      pry.Logger
	repository  CustomerRepository
	validator   PhoneRepository
	application ApplicationService
	phones      map[string]string
	token       string
}

func (m *botManagement) Setup(ctx context.Context) error {
	for phone, value := range m.phones {
		if value == "" {
			return fmt.Errorf("phone %s is empty", phone)
		}
	}
	if m.token == "" {
		return fmt.Errorf("telegram token is required")
	}

	keyboard := &models.ReplyKeyboardMarkup{
		Keyboard:       [][]models.KeyboardButton{{{Text: ApplicationShortcut}}, {{Text: EmergencyShortcut}}, {{Text: DispatcherShortcut}}, {{Text: GuardShortcut}}},
		IsPersistent:   true,
		ResizeKeyboard: true,
	}
	plateRe := regexp.MustCompile(`^\s*[а-яА-Яa-zA-Z]\d{3}[а-яА-Яa-zA-Z]{2}\d{2,3}[\s.,]*$`)
	phoneRe := regexp.MustCompile(`^\+\d{11}$`)

	var contactsBlockedBefore time.Time

	b, err := bot.New(m.token, bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}
		var response string
		defer func() {
			if response != "" {
				_, err := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      update.Message.From.ID,
					Text:        response,
					ReplyMarkup: keyboard,
				})
				if err != nil {
					m.logger.Error(err, pry.Ctx(ctx))
				}

			}
		}()
		var hasPlate bool
		switch msg := update.Message.Text; {
		case msg == StartShortcut:
			if phone, err := m.repository.PhoneForCustomer(ctx, update.Message.From.ID); err != nil {
				response = WelcomePhrase
			} else {
				response = fmt.Sprintf(WelcomeAgainPhrase, phone)
			}
		case msg == EmergencyShortcut, msg == DispatcherShortcut, msg == GuardShortcut:
			phone, ok := m.phones[msg]
			if !ok {
				response = ContactNotFoundPhrase
				m.logger.Error(fmt.Sprintf("No phone for %s", msg), pry.Ctx(ctx))
				return
			}
			if contactsBlockedBefore.After(time.Now()) {
				response = fmt.Sprintf("%s: %s\n%s", CondominiumName, msg, phone)
				return
			}
			_, err := b.SendContact(ctx, &bot.SendContactParams{
				ChatID:      update.Message.Chat.ID,
				PhoneNumber: phone,
				FirstName:   fmt.Sprintf("%s: %s", CondominiumName, msg),
				ReplyMarkup: keyboard,
			})
			if err != nil {
				if strings.HasPrefix(err.Error(), "unexpected response statusCode 429 for method sendContact, ") {
					m.logger.Warn(err, pry.Ctx(ctx))
					matches := regexp.MustCompile(`Too Many Requests: retry after (\d+)`).FindStringSubmatch(err.Error())
					duration, _ := strconv.Atoi(matches[1])
					contactsBlockedBefore = time.Now().Add(time.Second * time.Duration(duration))
					response = fmt.Sprintf("%s: %s\n%s", CondominiumName, msg, phone)
				} else {
					response = UnknownTelegramErrorPhrase
					m.logger.Error(err, pry.Ctx(ctx))
				}
			}
		case update.Message != nil && update.Message.Contact != nil:
			msg = update.Message.Contact.PhoneNumber
			fallthrough
		case phoneRe.MatchString(msg):
			if gates, err := m.validator.ValidatePhone(ctx, msg); err == nil && len(gates) > 0 {
				if err = m.repository.SaveCustomer(ctx, update.Message.From.ID, msg); err == nil {
					response = ReadyForApplicationPhrase
				} else {
					m.logger.Error(err, pry.Ctx(ctx))
					response = OopsPhrase
				}
			} else {
				response = NotFoundPhrase
			}
		case plateRe.MatchString(msg):
			msg = strings.Trim(update.Message.Text, ",. ")
			hasPlate = true
			fallthrough
		case msg == ApplicationShortcut:
			phone, err := m.repository.PhoneForCustomer(ctx, update.Message.From.ID)
			if err != nil {
				response = UnknownPersonPhrase
				return
			}
			gates, err := m.validator.ValidatePhone(ctx, phone)
			if err != nil || len(gates) == 0 {
				response = PhoneChangedPhrase
				return
			}
			if !hasPlate {
				response = WaitForPlatePhrase
				return
			}
			err = m.application.Application(ctx, phone, msg, gates)
			if err != nil {
				m.logger.Error(err, pry.Ctx(ctx))
				response = ApplicationFailedPhrase
				return
			}

			response = ApplicationSentPhrase
		default:
			response = UnknownPhrase
		}
	}))
	if err != nil {
		return err
	}

	b.Start(ctx)
	return nil
}
