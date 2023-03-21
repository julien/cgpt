package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	openAPIKey = "OPENAPI_KEY"
	url        = "https://api.openai.com/v1/chat/completions"
	usage      = "usage: cgpt PROMPT\nwhere PROMPT is the question you want to ask\n"
)

type (
	response struct {
		Choices []choices `json:"choices"`
	}

	choices struct {
		Message message `json:"message"`
	}

	message struct {
		Content string `json:"content"`
	}
)

func main() {
	var (
		prompt string
		key    string
		body   string
		reader *bytes.Reader
		req    *http.Request
		client http.Client
		res    *http.Response
		data   response
	)

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}

	prompt = strings.Join(os.Args[1:], " ")

	key = os.Getenv(openAPIKey)
	if len(key) == 0 {
		fmt.Fprintf(os.Stderr, "you need to set the %s environment variable before usage\n", openAPIKey)
		os.Exit(1)
	}

	body = `{
		"model": "gpt-3.5-turbo",
		"messages": [{"role": "user", "content": "` + prompt + `"}]
	}`
	reader = bytes.NewReader([]byte(body))
	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't make request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	go spinner(100 * time.Millisecond)

	client = http.Client{Timeout: 30 * time.Second}
	res, err = client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request error: %v\n", err)
		os.Exit(1)
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		fmt.Fprintf(os.Stderr, "couldn't read response: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", data.Choices[0].Message.Content)
	os.Exit(0)
}

func spinner(delay time.Duration) {
	for {
		for _, r := range `-\|/` {
			fmt.Printf("\r%c", r)
			time.Sleep(delay)
		}
	}
}
