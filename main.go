package main

import (
	"context"
	"errors"
	"fmt"
	"katyabot/e"
	"katyabot/storage"
	"katyabot/storage/files"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	storagePath, exists := os.LookupEnv("STORAGE_PATH")
	if !exists {
		log.Fatal(errors.New("storage path not exist"))
	}

	token, exists := os.LookupEnv("TELEGRAM_TOKEN")
	if !exists {
		log.Fatal(errors.New("token not exist"))
	}

	adminPass, exists := os.LookupEnv("ADMIN_PASS")
	if !exists {
		log.Fatal(errors.New("adminPass not exist"))
	}

	groupLink, exists := os.LookupEnv("GROUP_LINK")
	groupID, _ := strconv.Atoi(groupLink)
	if !exists {
		log.Fatal(errors.New("chat link not exist"))
	}

	adminID, exists := os.LookupEnv("ADMIN_USERNAME")
	if !exists {
		log.Fatal(errors.New("admin username not exist"))
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(messageHandler(files.New(storagePath), int64(groupID), adminPass, adminID)),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		panic(err)
	}

	log.Print("start server")
	b.Start(ctx)
}

func messageHandler(data files.Storage, groupID int64, adminPass string, adminID string) func(ctx context.Context, b *bot.Bot, update *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}
		username := update.Message.From.Username
		chatID := update.Message.Chat.ID

		if chatID == groupID {
			if update.Message.From.Username == adminID {
				if update.Message.Text != "" {
					words := strings.Split(strings.TrimSpace(update.Message.Text), " ")
					if len(words) >= 3 && words[0] == storage.MSG {
						u := words[1]

						exist, err := data.IsExists(u)
						if err != nil {
							msg := fmt.Sprintf("can't check existence: [%s]: ", username)
							log.Print(e.Wrap(msg, err))
							return
						}

						if !exist {
							sendTextMessage(ctx, b, groupID, storage.UNKNOWN_USER+u)
						} else {
							ud, _, err := data.LoadData(username)
							if err != nil {
								msg := fmt.Sprintf("can't load data: [%s]: ", username)
								log.Print(e.Wrap(msg, err))
								return
							}

							msg := strings.Join(words[2:], " ")
							sendTextMessage(ctx, b, ud.ChatID, msg+"\n\n(c) Большой брат")
						}
					}
				}
			}
			return
		}

		exists, err := data.IsExists(username)
		if err != nil {
			msg := fmt.Sprintf("can't check existence: [%s]: ", username)
			log.Print(e.Wrap(msg, err))
			return
		}

		ud, userExist, _ := data.LoadData(username)

		if !userExist {
			ud = storage.UserData{
				UserName:     username,
				NAME:         "",
				ChatID:       chatID,
				CurrentLevel: 0,
				Mode:         storage.User,
			}
		}

		switch update.Message.Text {
		case storage.START:
			if !exists {
				data.Save(ud)
				sendTextMessage(ctx, b, chatID, storage.HELLO+"\n\nПривет, @"+ud.UserName+"!")
				sendTextMessage(ctx, b, ud.ChatID, storage.FIRST_QUESTION)
			}

		case storage.ADMIN + adminPass:
			ud.Mode = storage.Admin
			sendTextMessage(ctx, b, chatID, storage.ADMIN_RESPONSE)

		case storage.DATA:
			if userExist && ud.Mode == storage.Admin {
				sendTextMessage(ctx, b, chatID, ud.ToString())
			}
			return

		case storage.CHECK:
			if userExist && ud.Mode == storage.Admin {
				checkAll(data, func(msg string) {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: ud.ChatID,
						Text:   msg,
					})
				})
			}
			return

		case storage.RESET:
			data.Remove(ud.UserName)
			return

		default:
			if ud.UserName == adminID {
				if update.Message.Voice != nil {
					sendTextMessage(ctx, b, ud.ChatID, update.Message.Voice.FileID)
				}

				if update.Message.Photo != nil {
					sendTextMessage(ctx, b, ud.ChatID, update.Message.Photo[len(update.Message.Photo)-1].FileID)
				}
			}
			switch ud.CurrentLevel {
			case 0:
				if update.Message.Photo != nil {
					sendTextMessage(ctx, b, chatID, `Принято! Болею за тебя!`)
					msg := fmt.Sprintf("Автор: @%s (#%s)\n#Задание%d", ud.UserName, ud.UserName, ud.CurrentLevel+1)
					sendPhotoMessage(ctx, b, groupID, update.Message.Photo[len(update.Message.Photo)-1].FileID, msg)
					ud.CurrentLevel += 1
					sendTextMessage(ctx, b, chatID, storage.SECOND_QUESTION)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на фото!\nЖду фото...")
				}

			case 1:
				if update.Message.Text != "" {
					sendTextMessage(ctx, b, chatID, `Принято! Так держать!`)
					msg := fmt.Sprintf("Автор: @%s (#%s)\n#Задание%d\n\n%s", ud.UserName, ud.UserName, ud.CurrentLevel+1, update.Message.Text)
					sendTextMessage(ctx, b, groupID, msg)
					ud.CurrentLevel += 1
					sendPhotoMessage(ctx, b, chatID, storage.REBUS_LINK, storage.THIRD_QUESTION)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на историю!\nЖду текстовое сообщение...")
				}

			case 2:
				if strings.EqualFold(update.Message.Text, "ПИВО!") {
					ud.CurrentLevel += 1
					sendTextMessage(ctx, b, chatID, storage.FOURTH_QUESTION)
					sendAudioMessage(ctx, b, chatID, storage.VOICE_LINK)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на ответ!\nЖду текстовое сообщение...")
				}

			case 3:
				if strings.EqualFold(update.Message.Text, "Настя") || strings.EqualFold(update.Message.Text, "Анастасия") || strings.EqualFold(update.Message.Text, "Настенька") { //////////////////??????????????????????????
					ud.CurrentLevel += 1
					sendTextMessage(ctx, b, chatID, `Принято! Молодец!`)
					sendTextMessage(ctx, b, chatID, storage.FIFTH_QUESTION)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на ответ!\nЖду текстовое сообщение с именем шпиона...")
				}

			case 4:
				if update.Message.Video != nil || update.Message.VideoNote != nil {
					if update.Message.Video != nil {
						msg := fmt.Sprintf("Автор: @%s (#%s)\n#Задание%d\n\n%s", ud.UserName, ud.UserName, ud.CurrentLevel+1, update.Message.Text)
						sendVideo(ctx, b, groupID, update.Message.Video.FileID, msg)
					} else if update.Message.VideoNote != nil {
						sendVideoNote(ctx, b, groupID, update.Message.VideoNote.FileID)
						msg := fmt.Sprintf("Автор: @%s (#%s)\n#Задание%d\n\n%s", ud.UserName, ud.UserName, ud.CurrentLevel+1, update.Message.Text)
						sendTextMessage(ctx, b, groupID, msg)
					}

					ud.CurrentLevel += 1
					sendTextMessage(ctx, b, chatID, "Какие же вы красивые😍😍😍\n\nПринято!")
					sendTextMessage(ctx, b, chatID, storage.SIXTH_QUESTION)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на ответ!\nЖду видео/кружок...")
				}

			case 5:
				if strings.EqualFold(update.Message.Text, "Подумаю об этом завтра") {
					ud.CurrentLevel += 1
					sendTextMessage(ctx, b, chatID, storage.SEVENTH_QUESTION)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на ответ!\nЖду текстовое сообщение с паролем от шпиона...")
				}

			case 6:
				if update.Message.Text != "" {
					sendTextMessage(ctx, b, chatID, `Принято! Так держать!`)
					msg := fmt.Sprintf("Автор: @%s (#%s)\n#Задание%d\n\n%s", ud.UserName, ud.UserName, ud.CurrentLevel+1, update.Message.Text)
					sendTextMessage(ctx, b, groupID, msg)
					ud.CurrentLevel += 1
					sendTextMessage(ctx, b, chatID, storage.QUESTION_TIME)
				} else {
					sendTextMessage(ctx, b, chatID, "Пупупу, это не похоже на историю!\nЖду текстовое сообщение...")
				}

			case 7:
				sendTextMessage(ctx, b, chatID, storage.QUESTION_TIME)
			}

		}

		log.Print(ud.UserName + " cmd: " + update.Message.Text)
		data.Save(ud)
	}
}

func syncFirstQuestion(ctx context.Context, b *bot.Bot, chatID int64) {
	go func() {
		time.Sleep(time.Second * 2)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   storage.WARNING,
		})

		time.Sleep(time.Second * 2)

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   storage.FIRST_QUESTION,
		})
	}()
}

func sendTextMessage(ctx context.Context, b *bot.Bot, chatID int64, msg string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   msg,
	})
}

func sendAudioMessage(ctx context.Context, b *bot.Bot, chatID int64, voiceLink string) {
	b.SendVoice(ctx, &bot.SendVoiceParams{
		ChatID: chatID,
		Voice:  &models.InputFileString{Data: voiceLink},
	})
}

func sendVideoNote(ctx context.Context, b *bot.Bot, chatID int64, videoNoteLink string) {
	b.SendVideoNote(ctx, &bot.SendVideoNoteParams{
		ChatID:    chatID,
		VideoNote: &models.InputFileString{Data: videoNoteLink},
	})
}

func sendVideo(ctx context.Context, b *bot.Bot, chatID int64, videoLink string, msg string) {
	b.SendVideo(ctx, &bot.SendVideoParams{
		ChatID:  chatID,
		Video:   &models.InputFileString{Data: videoLink},
		Caption: msg,
	})
}

func sendPhotoMessage(ctx context.Context, b *bot.Bot, chatID int64, img string, caption string) {
	params := &bot.SendPhotoParams{
		ChatID:         chatID,
		ProtectContent: false,
		Caption:        caption,
		Photo:          &models.InputFileString{Data: img},
	}
	b.SendPhoto(ctx, params)
}

func sendMediaGroup(ctx context.Context, b *bot.Bot, chatID int64, media []models.InputMedia) {
	params := &bot.SendMediaGroupParams{
		ChatID:         chatID,
		Media:          media,
		ProtectContent: false,
	}

	b.SendMediaGroup(ctx, params)
}

func checkAll(data files.Storage, send func(msg string)) {
	files, err := os.ReadDir(data.BasePath)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		ud, _, err := data.DecodeFile(filepath.Join(data.BasePath, f.Name()))
		if err != nil {
			log.Fatal(err)
		}

		send(ud.ToString())
	}
}
