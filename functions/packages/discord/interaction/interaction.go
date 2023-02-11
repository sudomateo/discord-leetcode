package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/go-hclog"
	"github.com/sudomateo/discord-leetcode/functions/packages/discord/interaction/leetcode"
)

// Response is the response the DigitalOcean functions expects in order to send
// responses to clients.
type Response struct {
	StatusCode int               `json:"statusCode,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       interface{}       `json:"body,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type PingResponse struct {
	Type discordgo.InteractionType `json:"type"`
}

func HandleInteraction(args map[string]interface{}) *Response {
	log := hclog.New(&hclog.LoggerOptions{
		Name: "discord-leetcode",
	})

	defer func() {
		if v := recover(); v != nil {
			log.Error("PANIC", "value", v)
		}
	}()

	log.Info("request received")
	defer log.Info("request complete")

	r := requestFromArgs(args)

	publicKey := os.Getenv("DISCORD_APP_PUBLIC_KEY")
	if publicKey == "" {
		log.Error("missing discord application public key")
		return &Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		log.Error("could not decode public key", "error", err)
		return &Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	if !discordgo.VerifyInteraction(r, ed25519.PublicKey(publicKeyBytes)) {
		log.Error("interaction failed verification")
		return &Response{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusBadRequest),
			},
		}
	}

	var interaction discordgo.Interaction
	if err := json.NewDecoder(r.Body).Decode(&interaction); err != nil {
		log.Error("invalid interation payload", "error", err)
		return &Response{
			StatusCode: http.StatusBadRequest,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusBadRequest),
			},
		}
	}

	if interaction.Type == discordgo.InteractionType(discordgo.InteractionResponsePong) {
		log.Info("acknowledging interaction", "type", interaction.Type)
		return &Response{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: PingResponse{
				Type: discordgo.InteractionType(discordgo.InteractionResponsePong),
			},
		}
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Error("missing discord token")
		return &Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Error("could not create discord client", "error", err)
		return &Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	lc := leetcode.Client{}
	resp, err := lc.RandomQuestion(leetcode.RandomDifficulty())
	if err != nil {
		log.Error("could not retrieve leetcode question", "error", err)
		return &Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	interactionResp := discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("https://leetcode.com/problems/%s", resp.Data.RandomQuestion.TitleSlug),
		},
	}

	if err := d.InteractionRespond(&interaction, &interactionResp); err != nil {
		log.Error("could not response to interaction", "error", err, "type", interaction.Type)
		return &Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: ErrorResponse{
				Error: http.StatusText(http.StatusInternalServerError),
			},
		}
	}

	return nil
}

func requestFromArgs(args map[string]interface{}) *http.Request {
	r := http.Request{
		Header: make(http.Header),
	}

	if http, ok := args["http"].(map[string]interface{}); ok {
		if headerMap, ok := http["headers"].(map[string]interface{}); ok {
			for header, valueMap := range headerMap {
				if value, ok := valueMap.(string); ok {
					r.Header.Set(header, value)
				}
			}
		}
	}

	if http, ok := args["http"].(map[string]interface{}); ok {
		if body, ok := http["body"].(string); ok {
			reader := strings.NewReader(body)
			r.Body = io.NopCloser(reader)
		}
	}

	return &r
}
