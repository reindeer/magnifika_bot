package bot

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

const (
	CondominiumName = "Магнифика"

	StartShortcut       = "/start"
	ApplicationShortcut = "Заявка на въезд"
	EmergencyShortcut   = "Аварийно-диспетчерская служба"
	DispatcherShortcut  = "Заявка на въезд по телефону"
	GuardShortcut       = "Охрана"

	WelcomePhrase             = "Привет! Давай познакомимся! Пришли мне свой номер телефона, зарегистрированный в УК, в формате +7xxxxxxxxxx.\nНо имей в виду, что я запомню твой телефон. Его сможет увидеть только папа @tarandro (если захочет)."
	WelcomeAgainPhrase        = "Привет, рад видеть тебя снова! Если у тебя новый телефон, пришли мне его. Сейчас у меня записан: %s"
	ReadyForApplicationPhrase = "Готово! Теперь можешь создавать заявки."
	OopsPhrase                = "Ой, кажется я поломался! Позовите папу @tarandro!"
	NotFoundPhrase            = "УК не может найти тебя в списках. Свяжись с ними или укажи телефон, который зарегистрирован в УК."
	PhoneChangedPhrase        = "Что-то пошло не так и УК перестала тебя узнавать. Свяжись с УК или пришли мне свой новый номер телефона."
	UnknownPersonPhrase       = "Извини, папа запрещает мне разговаривать с незнакомцами. Пришли мне свой номер телефона, зарегистрированный в УК."
	UnknownPhrase             = "Ой, что-то я не понял тебя. Что ты имеешь в виду?"
	ApplicationFailedPhrase   = "Все было хорошо, но УК твою заявку не приняла. Не знаю, почему. Спроси папу @tarandro, он знает."
	ApplicationSentPhrase     = "Готово! Заявку отправил.\nВъезжать можно только с Магнитогорской улицы."
	WaitForPlatePhrase        = "Скажи, кого надо пропустить, и я передам дальше.\nМне нужен полный номер с регионом.\nНапример а000аа78."
)

var keyboard = []string{ApplicationShortcut, EmergencyShortcut, DispatcherShortcut, GuardShortcut}

type reply struct {
	text    string
	contact *model.ContactCard
}

func (b *bot) Handle(ctx context.Context, message model.Message) error {
	r := b.answer(ctx, message)
	if r.contact != nil {
		err := b.messenger.SendContact(ctx, message.ChatId, *r.contact, keyboard)
		if err == nil {
			return nil
		}
		if !errors.Is(err, model.ErrRateLimited) {
			b.logger.Error(err, pry.Ctx(ctx))
		}
		r.text = r.contact.Name + "\n" + r.contact.Phone
	}
	return b.messenger.SendText(ctx, message.ChatId, r.text, keyboard)
}

func (b *bot) answer(ctx context.Context, message model.Message) reply {
	text := message.Text
	contactPhone, isContact := b.contacts[text]
	plate, isPlate := model.ParsePlate(text)
	switch {
	case text == StartShortcut:
		return b.replyStart(ctx, message.CustomerId)
	case isContact:
		return reply{contact: &model.ContactCard{Name: fmt.Sprintf("%s: %s", CondominiumName, text), Phone: contactPhone}}
	case message.SharedPhone != "":
		return b.replyRegister(ctx, message.CustomerId, message.SharedPhone)
	case model.IsPhone(text):
		return b.replyRegister(ctx, message.CustomerId, text)
	case isPlate:
		return b.replyApplicant(ctx, b.apply(ctx, message.CustomerId, plate), ApplicationSentPhrase, ApplicationFailedPhrase)
	case text == ApplicationShortcut:
		_, _, err := b.applicant(ctx, message.CustomerId)
		return b.replyApplicant(ctx, err, WaitForPlatePhrase, OopsPhrase)
	default:
		return reply{text: UnknownPhrase}
	}
}

func (b *bot) replyStart(ctx context.Context, customerId int64) reply {
	phone, err := b.phone(ctx, customerId)
	if errors.Is(err, model.ErrUnknownCustomer) {
		return reply{text: WelcomePhrase}
	}
	if err != nil {
		b.logger.Error(err, pry.Ctx(ctx))
		return reply{text: OopsPhrase}
	}
	return reply{text: fmt.Sprintf(WelcomeAgainPhrase, phone)}
}

func (b *bot) replyRegister(ctx context.Context, customerId int64, phone string) reply {
	err := b.register(ctx, customerId, phone)
	if errors.Is(err, model.ErrUnknownPhone) {
		return reply{text: NotFoundPhrase}
	}
	if err != nil {
		b.logger.Error(err, pry.Ctx(ctx))
		return reply{text: OopsPhrase}
	}
	return reply{text: ReadyForApplicationPhrase}
}

func (b *bot) replyApplicant(ctx context.Context, err error, success, failure string) reply {
	switch {
	case err == nil:
		return reply{text: success}
	case errors.Is(err, model.ErrUnknownCustomer):
		return reply{text: UnknownPersonPhrase}
	case errors.Is(err, model.ErrPhoneChanged):
		return reply{text: PhoneChangedPhrase}
	default:
		b.logger.Error(err, pry.Ctx(ctx))
		return reply{text: failure}
	}
}
