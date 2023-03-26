package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	key = "OPENAI_API_KEY"
	url = "https://api.openai.com/v1/chat/completions"
)

var (
	errKeyNotSet      = errors.New("the OPENAI_API_KEY environment variable is not set")
	errInvalidInput   = errors.New("couldn't scan user input")
	errInvalidPayload = errors.New("couldn't generate payload")
	errNoResults      = errors.New("couldn't fetch results")
)

type (
	body struct {
		Messages *[]message `json:"messages"`
		Model    string     `json:"model"`
	}

	response struct {
		ID      string    `json:"id"`
		Object  string    `json:"object"`
		Model   string    `json:"model"`
		Choices []choices `json:"choices"`
		Usage   usage     `json:"usage"`
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

	// an interface to make it easier to mock http.Client
	httpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	config struct {
		client httpClient
		ctx    context.Context
		input  io.Reader
		output io.Writer
		key    string
	}
)

func main() {
	if err := run(config{
		client: http.DefaultClient,
		ctx:    context.Background(),
		key:    os.Getenv(key),
		input:  os.Stdin,
		output: os.Stdout,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	if len(cfg.key) == 0 {
		return errKeyNotSet
	}
	return loop(cfg)
}

func loop(cfg config) error {
	var (
		msgs = make([]message, 0, 2)
		quit = make(chan struct{})
	)

	fmt.Fprintln(cfg.output, "enter your question, and type ENTER")

	for {
		fmt.Fprint(cfg.output, "> ")
		txt, err := input(cfg.input)
		if err != nil {
			return errInvalidInput
		}

		b, err := payload(&msgs, "user", txt)
		if err != nil {
			return errInvalidPayload
		}

		go spinner(cfg.ctx, 100*time.Millisecond, quit)
		resp, err := request(cfg.client, b, cfg.key)
		quit <- struct{}{}
		if err != nil {
			return errNoResults
		}

		c := resp.Choices[0].Message.Content
		fmt.Fprintf(cfg.output, "%s\n\n", c)

		_, err = payload(&msgs, resp.Choices[0].Message.Role, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "couldn't update conversation: %v\n", err)
		}

		select {
		case <-cfg.ctx.Done():
			return nil
		default:
			continue
		}

	}
	return nil
}

func input(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return scanner.Text(), nil
}

func payload(msgs *[]message, role, input string) ([]byte, error) {
	var (
		msg = message{Role: role, Content: input}
		b   body
		s   []byte
		err error
	)

	*msgs = append(*msgs, msg)
	b = body{Model: "gpt-3.5-turbo", Messages: msgs}
	s, err = json.Marshal(&b)
	if err != nil {
		return nil, err
	}
	return s, err
}

func request(client httpClient, payload []byte, key string) (*response, error) {
	var (
		req  *http.Request
		res  *http.Response
		data response
		err  error
	)

	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	res, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func spinner(ctx context.Context, delay time.Duration, quit <-chan struct{}) {
	t := time.NewTicker(delay)
	for {
		select {
		case <-ctx.Done():
		case <-quit:
			return
		case <-t.C:
			for _, r := range `-\|/` {
				fmt.Printf("\r%c", r)
				time.Sleep(delay / 2)
			}
		}
	}
}
