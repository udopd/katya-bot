package files

import (
	"encoding/gob"
	"errors"
	"fmt"
	"katyabot/e"
	"katyabot/storage"
	"os"
	"path/filepath"
)

type Storage struct {
	BasePath string
}

func New(basePath string) Storage {
	return Storage{BasePath: basePath}
}

func (s Storage) Save(ud storage.UserData) (err error) {
	fName, err := storage.Hash(ud.UserName)
	if err != nil {
		return e.Wrap("can't save data", err)
	}
	fName += ".ud"
	fPath := filepath.Join(s.BasePath, fName)

	file, err := os.Create(fPath)
	if err != nil {
		return e.Wrap("can't save data", err)
	}
	defer func() { _ = file.Close() }()

	if err := gob.NewEncoder(file).Encode(ud); err != nil {
		return e.Wrap("can't encode file", err)
	}

	return nil
}

func (s Storage) LoadData(userName string) (ud storage.UserData, exist bool, err error) {
	fName, err := storage.Hash(userName)
	if err != nil {
		return ud, false, e.Wrap("can't hash filename", err)
	}
	fName += ".ud"
	fPath := filepath.Join(s.BasePath, fName)

	return s.DecodeFile(fPath)
}

func (s Storage) Remove(userName string) error {
	fName, err := storage.Hash(userName)
	if err != nil {
		return e.Wrap("can't hash filename", err)
	}
	fName += ".ud"
	fPath := filepath.Join(s.BasePath, fName)

	if err := os.Remove(fPath); err != nil {
		msg := fmt.Sprintf("can't remove file %s", fPath)
		return e.Wrap(msg, err)
	}

	return nil
}

func (s Storage) IsExists(username string) (bool, error) {
	fName, err := storage.Hash(username)
	if err != nil {
		return false, e.Wrap("can't check if file exists", err)
	}
	fName += ".ud"
	fPath := filepath.Join(s.BasePath, fName)

	switch _, err = os.Stat(fPath); {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		msg := fmt.Sprintf("can't check if file %s exists", fPath)
		return false, e.Wrap(msg, err)
	}

	return true, nil
}

func (s Storage) DecodeFile(filePath string) (storage.UserData, bool, error) {
	ud := storage.UserData{}
	f, err := os.Open(filePath)
	if err != nil {
		return ud, false, e.Wrap("file not exist", err)
	}
	defer func() { _ = f.Close() }()

	if err := gob.NewDecoder(f).Decode(&ud); err != nil {
		return ud, false, e.Wrap("can't decode file", err)
	}

	return ud, true, nil
}
