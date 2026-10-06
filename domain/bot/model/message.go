package model

type Message struct {
	ChatId      int64
	CustomerId  int64
	Text        string
	SharedPhone string
}

type ContactCard struct {
	Name  string
	Phone string
}
