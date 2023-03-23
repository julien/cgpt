package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	openAPIKey = "OPENAPI_KEY"
	url        = "https://api.openai.com/v1/chat/completions"
)

type (
	body struct {
		Model    string     `json:"model"`
		Messages *[]message `json:"messages"`
	}

	response struct {
		ID      string    `json:"id"`
		Object  string    `json:"object"`
		Model   string    `json:"model"`
		Usage   usage     `json:"usage"`
		Choices []choices `json:"choices"`
	}

	usage struct {
		PromptTokens     int32 `json:"prompt_tokens"`
		CompletionTokens int32 `json:"completion_tokens"`
		TotalTokens      int32 `json:"total_tokens"`
	}

	choices struct {
		Message      message `json:"message"`
		FinishReason string  `json:"finish_reason"`
		Index        int     `json:"index"`
	}

	message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	key := os.Getenv(openAPIKey)
	if len(key) == 0 {
		return errors.New("the environment variable OPENAI_KEY is not set")
	}
	return loop(key)
}

func loop(key string) error {
	msgs := make([]message, 0, 2)
	quit := make(chan struct{})
	fmt.Println("enter your question, and type ENTER")

	for {
		txt, err := input()
		if err != nil {
			return errors.New("couldn't scan user input")
		}

		b, err := payload(&msgs, "user", txt)
		if err != nil {
			return errors.New("couldn't generate payload")
		}

		req, err := request(b, key)

		if err != nil {
			return errors.New("couldn't generate request")
		}

		go spinner(100*time.Millisecond, quit)
		resp, err := fetch(req)
		quit <- struct{}{}
		if err != nil {
			return errors.New("couldn't fetch results")
		}

		c := resp.Choices[0].Message.Content
		fmt.Printf("%s\n\n", c)

		b, err = payload(&msgs, resp.Choices[0].Message.Role, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't update conversation: %v\n", err)
			break
		}
	}
	return nil
}

func input() (string, error) {
	fmt.Print("> ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return scanner.Text(), nil
}

func payload(msgs *[]message, role, input string) ([]byte, error) {
	var (
		msg message
		b   body
		s   []byte
		err error
	)

	msg = message{Role: "user", Content: input}
	*msgs = append(*msgs, msg)
	b = body{Model: "gpt-3.5-turbo", Messages: msgs}
	s, err = json.Marshal(&b)
	if err != nil {
		return nil, err
	}
	return s, err
}

func request(payload []byte, key string) (*http.Request, error) {
	var (
		reader *bytes.Reader
		req    *http.Request
		err    error
	)

	reader = bytes.NewReader(payload)
	req, err = http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func fetch(req *http.Request) (*response, error) {
	var (
		client http.Client
		res    *http.Response
		data   response
		err    error
	)

	client = http.Client{Timeout: 2 * time.Minute}
	res, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func spinner(delay time.Duration, quit <-chan struct{}) {
	t := time.NewTicker(delay)
	for {
		select {
		case <-quit:
			return
		case <-t.C:
			for _, r := range `-\|/` {
				fmt.Printf("\r%c", r)
				time.Sleep(delay)
			}
		}
	}
}
