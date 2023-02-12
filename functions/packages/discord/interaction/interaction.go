package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// ErrorResponse is the response we send to clients when an error occurs.
type ErrorResponse struct {
	Error string `json:"error"`
}

// PingResponse is the response sent to acknowledge a Discord ping.
type PingResponse struct {
	Type discordgo.InteractionType `json:"type"`
}

// InteractionData is the data we expect from the client.
type InteractionData struct {
	Data discordgo.ApplicationCommandInteractionData `json:"data"`
}

// HandleInteraction is the main entrypoint for this DigitalOcean function.
func HandleInteraction(args map[string]interface{}) *Response {
	log := hclog.New(&hclog.LoggerOptions{
		Name: "discord-leetcode",
	})

	defer func() {
		if v := recover(); v != nil {
			log.Error("panic", "value", v)
		}
	}()

	log.Info("request received")
	defer log.Info("request complete")

	r := parseRequest(args)

	if err := verifyRequestSignature(r); err != nil {
		log.Error("request verification failed", "error", err)
		return respondError(http.StatusUnauthorized)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r.Body); err != nil {
		log.Error("could not read request body", "error", err)
		return respondError(http.StatusInternalServerError)
	}
	body := bytes.NewReader(buf.Bytes())

	var interaction discordgo.Interaction
	if err := json.NewDecoder(body).Decode(&interaction); err != nil {
		log.Error("invalid interaction payload", "error", err)
		return respondError(http.StatusBadRequest)
	}

	// It's a ping request. Acknowledge it and respond with a pong.
	if interaction.Type == discordgo.InteractionType(discordgo.InteractionResponsePong) {
		log.Info("acknowledging interaction", "type", interaction.Type)
		return respond(http.StatusOK, PingResponse{
			Type: discordgo.InteractionType(discordgo.InteractionResponsePong),
		})
	}

	// We only support interactions of type application command.
	if interaction.Data.Type() != discordgo.InteractionType(discordgo.InteractionApplicationCommand) {
		log.Error("unsupported request type", "type", interaction.Data.Type().String())
		return respondError(http.StatusBadRequest)
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Error("missing discord token")
		return respondError(http.StatusInternalServerError)
	}

	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Error("could not create discord client", "error", err)
		return respondError(http.StatusInternalServerError)
	}

	// Seek back to the beginning of the request body so we can parse it into
	// the correct interaction type.
	if _, err := body.Seek(io.SeekStart, io.SeekStart); err != nil {
		log.Error("could not seek body", "error", err)
		return respondError(http.StatusInternalServerError)
	}

	var interactionData InteractionData
	if err := json.NewDecoder(body).Decode(&interactionData); err != nil {
		log.Error("invalid interation payload", "error", err, "type", interaction.Data.Type().String())
		return respondError(http.StatusBadRequest)
	}

	lcResp, err := fetchLeetCodeProblem(interactionData)
	if err != nil {
		log.Error("could not fetch leetcode problem", "error", err)
		return respondError(http.StatusInternalServerError)
	}

	interactionResp := discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("https://leetcode.com/problems/%s", lcResp.Data.RandomQuestion.TitleSlug),
		},
	}

	log.Info("responding to interaction",
		"type", interaction.Data.Type().String(),
		"id", interactionData.Data.ID,
		"name", interactionData.Data.Name,
		"target_id", interactionData.Data.TargetID,
	)

	if err := d.InteractionRespond(&interaction, &interactionResp); err != nil {
		log.Error("failed responding to interaction",
			"error", err,
			"type", interaction.Data.Type().String(),
			"id", interactionData.Data.ID,
			"name", interactionData.Data.Name,
			"target_id", interactionData.Data.TargetID,
		)
		return respondError(http.StatusInternalServerError)
	}

	return nil
}

// parseRequest parses the  request arguments into the Go representation of an
// HTTP request.
func parseRequest(args map[string]interface{}) *http.Request {
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
			r.Body = io.NopCloser(strings.NewReader(body))
		}
	}

	return &r
}

// verifyRequestSignature verifies whether or not the request signature is a
// valid Discord request.
func verifyRequestSignature(r *http.Request) error {
	publicKey := os.Getenv("DISCORD_APP_PUBLIC_KEY")
	if publicKey == "" {
		return errors.New("missing discord application public key")
	}

	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("invalid discord application public key: %w", err)
	}

	if !discordgo.VerifyInteraction(r, ed25519.PublicKey(publicKeyBytes)) {
		return errors.New("invalid request signature")
	}

	return nil
}

// fetchLeetCodeProblem retrieves a random LeetCode problem based on the
// interaction data.
func fetchLeetCodeProblem(interactionData InteractionData) (leetcode.RandomQuestionResponse, error) {
	var optDifficulty string

	for _, v := range interactionData.Data.Options {
		if v.Name == "difficulty" {
			optDifficulty = v.StringValue()
			break
		}
	}

	var difficulty leetcode.Difficulty

	switch leetcode.Difficulty(strings.ToUpper(optDifficulty)) {
	case leetcode.DifficultyEasy:
		difficulty = leetcode.DifficultyEasy
	case leetcode.DifficultyMedium:
		difficulty = leetcode.DifficultyMedium
	case leetcode.DifficultyHard:
		difficulty = leetcode.DifficultyHard
	default:
		difficulty = leetcode.RandomDifficulty()
	}

	lc := leetcode.NewClient()
	return lc.RandomQuestion(difficulty)
}

// respondError crafts an error response.
func respondError(statusCode int) *Response {
	return respond(statusCode, ErrorResponse{
		Error: http.StatusText(statusCode),
	})
}

// respond crafts a generic response.
func respond(statusCode int, body interface{}) *Response {
	return &Response{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
}
