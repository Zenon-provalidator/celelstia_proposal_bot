package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

type Config struct {
	API struct {
		URL string `yaml:"url"`
	} `yaml:"api"`
	Explorer struct {
		URL string `yaml:"url"`
	} `yaml:"explorer"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DBIndex  int    `yaml:"db_index"`
	} `yaml:"redis"`
	Telegram struct {
		BotToken string `yaml:"bot_token"`
		ChatID   int64  `yaml:"chat_id"`
	} `yaml:"telegram"`
	Ticker struct {
		Interval string `yaml:"interval"`
	} `yaml:"ticker"`
}

type ProposalResponse struct {
	Proposals []Proposal `json:"proposals"`
}

type Proposal struct {
	ProposalID string `json:"proposal_id"`
	Content    struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"content"`
}

func loadConfig(filename string) (*Config, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filename, err)
	}

	return c, nil
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       config.Redis.DBIndex,
	})

	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(config.Telegram.BotToken)
	if err != nil {
		log.Panic(err)
	}

	// Parse ticker interval
	interval, err := time.ParseDuration(config.Ticker.Interval)
	if err != nil {
		log.Fatalf("Error parsing ticker interval: %v", err)
	}

	// Loop
	quit := make(chan struct{})
	go func() {
		for {
			// Execute
			processProposals(config, rdb, bot)
			// Wait
			time.Sleep(interval)
		}
	}()
	<-quit
}

func processProposals(config *Config, rdb *redis.Client, bot *tgbotapi.BotAPI) {
	// Fetch proposals from API
	log.Printf("Call API")
	proposals, err := fetchProposals(config.API.URL)
	if err != nil {
		log.Printf("Error fetching proposals: %v", err)
		return
	}

	ctx := context.Background()

	for _, proposal := range proposals {
		key := fmt.Sprintf("proposal:%s", proposal.ProposalID)

		// Check if proposal exists in Redis
		exists, err := rdb.Exists(ctx, key).Result()
		if err != nil {
			log.Printf("Error checking Redis: %v", err)
			continue
		}

		if exists == 0 {
			// New proposal, store in Redis and send notification
			err = storeProposal(rdb, ctx, key, proposal)
			if err != nil {
				log.Printf("Error storing proposal: %v", err)
				continue
			}

			sendTelegramNotification(bot, config.Telegram.ChatID, proposal, config.Explorer.URL)
		} else {
			// Proposal exists, update in Redis
			err = storeProposal(rdb, ctx, key, proposal)
			if err != nil {
				log.Printf("Error updating proposal: %v", err)
			}
		}
	}
}

func fetchProposals(apiURL string) ([]Proposal, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var proposalResp ProposalResponse
	err = json.Unmarshal(body, &proposalResp)
	if err != nil {
		return nil, err
	}

	return proposalResp.Proposals, nil
}

func storeProposal(rdb *redis.Client, ctx context.Context, key string, proposal Proposal) error {
	proposalJSON, err := json.Marshal(proposal)
	if err != nil {
		return err
	}

	return rdb.Set(ctx, key, proposalJSON, 0).Err()
}

func sendTelegramNotification(bot *tgbotapi.BotAPI, chatID int64, proposal Proposal, expUrl string) {
	message := fmt.Sprintf("New Proposal:\nID: %s\nTitle: %s\nDescription: %s\nExplorer: %s/%s",
		proposal.ProposalID, proposal.Content.Title, proposal.Content.Description, expUrl, proposal.ProposalID)

	msg := tgbotapi.NewMessage(chatID, message)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending Telegram message: %v", err)
	}
}
