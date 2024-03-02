package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AppConfig struct {
	Token   string `json:"token"`
	OwnerID int64  `json:"ownerID"`
}

type APIResponse struct {
	IP string `json:"ip"`
}

func ReadAppConfig() AppConfig {
	configFile, err := os.Open("config.json")

	if err != nil {
		log.Fatalln(err)
	}
	defer configFile.Close()

	value, _ := io.ReadAll(configFile)

	var config AppConfig
	json.Unmarshal(value, &config)

	if config.OwnerID == 0 {
		log.Fatalln("Owner ID is missing or incorrect in config.json")
	}

	if config.Token == "" {
		log.Fatalln("Telegram token is missing or incorrect in config.json")
	}

	return config
}

func GetProviderIP() string {
	req, err := http.Get("https://api.ipify.org?format=json")

	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	var response APIResponse
	json.Unmarshal(body, &response)
	return response.IP
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return err.Error()
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func main() {
	appConfig := ReadAppConfig()

	bot, err := tgbotapi.NewBotAPI(appConfig.Token)

	if err != nil {
		log.Fatalln(err)
	}

	// bot.Debug = true

	cmdConfig := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{
			Command:     "ip",
			Description: "Get Local IP",
		},
		tgbotapi.BotCommand{
			Command:     "remote",
			Description: "Get Public IP",
		},
	)

	bot.Send(cmdConfig)

	log.Printf("Authorized on account %s", bot.Self.UserName)

	config := tgbotapi.NewUpdate(0)
	config.Timeout = 60

	updates := bot.GetUpdatesChan(config)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		if update.Message.From.ID != appConfig.OwnerID {
			log.Printf("[Unauthorized] [%s] %s", update.Message.From.UserName, update.Message.Text)
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

		switch update.Message.Command() {
		case "ip":
			msg.Text = fmt.Sprintf("Your local IP is %s", GetOutboundIP())
		case "remote":
			msg.Text = fmt.Sprintf("Your remote IP is %s", GetProviderIP())
		default:
			continue
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
