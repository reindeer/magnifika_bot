package management

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
	EmergenceShortcut   = "Аварийная служба"
	DispatcherShortcut  = "Диспетчер"
	GuardShortcut       = "Охрана"

	WelcomePhrase              = "Привет! Давай познакомимся! Пришли мне свой номер телефона, зарегистрированный в УК."
	WelcomeAgainPhrase         = "Привет, рад видеть тебя снова! Если у тебя новый телефон, пришли мне его. Сейчас у меня записан"
	ReadyForApplicationPhrase  = "Готово! Теперь можешь создавать заявки."
	OopsPhrase                 = "Ой, кажется я поломался! Позовите папу @tarandro!"
	NotFoundPhrase             = "УК не может найти тебя в списках. Свяжись с ними или укажи телефон, который зарегистрирован в УК."
	PhoneChangedPhrase         = "Что-то пошло не так и УК перестала тебя узнавать. Свяжись с УК или пришли мне свой новый номер телефона."
	ContactNotFoundPhrase      = "Что-то я потерял номер. Можешь попросить папу @tarandro помочь?"
	UnknownPhrase              = "Извини, папа запрещает мне разговаривать с незнакомцами. Пришли мне свой номер телефона, зарегистрированный в УК."
	UnknownTelegramErrorPhrase = "Telegram говорит что-то мне непонятное. Спроси папу @tarandro, может быть от знает."
	ApplicationFailedPhrase    = "Все было хорошо, но УК твою заявку не приняла. Не знаю, почему. Спроси папу @tarandro, он знает."
	ApplicationSentPhrase      = "Готово! Заявку отправил."
	WaitForPlatePhrase         = "Скажи, кого надо пропустить, и я передам дальше.\nМне нужен полный номер с регионом.\nНапример а000аа78."
)

type CustomerRepository interface {
	PhoneForCustomer(ctx context.Context, id int64) (string, error)
	SaveCustomer(ctx context.Context, id int64, phone string) error
}

type PhoneAdapter interface {
	Init(ctx context.Context) error
	ValidatePhone(ctx context.Context, phone string) ([]string, error)
}

type ApplicationAdapter interface {
	Init(ctx context.Context) error
	Application(ctx context.Context, phone, plate string, gates []string) error
}

type Registry interface {
	Get(ctx context.Context, code string) (string, error)
}

func NewBotManagement(
	registry Registry,
	repository CustomerRepository,
	validator PhoneAdapter,
	application ApplicationAdapter,
	logger pry.Logger,
) (*botManagement, error) {
	return &botManagement{
		logger:      logger,
		repository:  repository,
		validator:   validator,
		application: application,
		registry:    registry,
	}, nil
}

type botManagement struct {
	logger      pry.Logger
	repository  CustomerRepository
	validator   PhoneAdapter
	application ApplicationAdapter
	registry    Registry
	phones      map[string]string
	token       string
}

func (m *botManagement) Setup(ctx context.Context) error {
	err := m.init(ctx)
	if err != nil {
		return err
	}

	keyboard := &models.ReplyKeyboardMarkup{
		Keyboard:       [][]models.KeyboardButton{{{Text: ApplicationShortcut}}, {{Text: EmergenceShortcut}}, {{Text: DispatcherShortcut}}, {{Text: GuardShortcut}}},
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
				response = fmt.Sprintf("%s: %s", WelcomeAgainPhrase, phone)
			}
		case msg == EmergenceShortcut, msg == DispatcherShortcut, msg == GuardShortcut:
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
				response = UnknownPhrase
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

			response = fmt.Sprintf("%s\nВъезжать можно через ворота «%s»", ApplicationSentPhrase, strings.Join(gates, "» или «"))
			return
		default:
			m.logger.Trace("Skip unknown message: " + msg)
		}
	}))
	if err != nil {
		return err
	}

	b.Start(ctx)
	return nil
}

func (m *botManagement) init(ctx context.Context) error {
	err := m.application.Init(ctx)
	if err != nil {
		return err
	}
	err = m.validator.Init(ctx)
	if err != nil {
		return err
	}

	m.phones = make(map[string]string, 3)
	m.phones[EmergenceShortcut], err = m.registry.Get(ctx, EmergenceShortcut)
	if err != nil {
		return err
	}
	m.phones[DispatcherShortcut], err = m.registry.Get(ctx, DispatcherShortcut)
	if err != nil {
		return err
	}

	m.phones[GuardShortcut], err = m.registry.Get(ctx, GuardShortcut)
	if err != nil {
		return err
	}

	m.token, err = m.registry.Get(ctx, "telegram.token")
	if err != nil {
		return err
	}

	return nil
}
