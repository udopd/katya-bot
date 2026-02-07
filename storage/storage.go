package storage

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"katyabot/e"
)

type Storage interface {
	Save(ud *UserData) error
	LoadData(userName string) (*UserData, error)
	Remove(userName string) error
	IsExists(ud *UserData) (bool, error)
}

var ErrNoSavedFiles = errors.New("no saved files")

func Hash(str string) (string, error) {
	h := sha1.New()

	if _, err := io.WriteString(h, str); err != nil {
		return "", e.Wrap("can't calculate hash", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
