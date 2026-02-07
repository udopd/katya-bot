package storage

import (
	"fmt"
)

type Mode string

const (
	Admin Mode = "admin"
	User  Mode = "user"
)

type UserData struct {
	UserName string
	ChatID   int64
	NAME     string
	Mode     Mode

	CurrentLevel int
}

func (u *UserData) ToString() (str string) {
	str = fmt.Sprintf("username: @%s\n", u.UserName)
	str += fmt.Sprintf("chatID: %d\n", u.ChatID)
	str += fmt.Sprintf("NAME: %s\n", u.NAME)

	str += fmt.Sprintf("mode: %s\n", u.Mode)
	str += fmt.Sprintf("level: %d\n", u.CurrentLevel)
	return str
}
