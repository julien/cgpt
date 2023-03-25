package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInput(t *testing.T) {
	tcs := []struct {
		name string
		s    string
		fail bool
	}{
		{
			name: "returns a string given valid input",
			s:    "hello",
		},
		{
			name: "handles special characters '\n'",
			s:    "\n",
			fail: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.s)
			txt, err := input(r)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if txt != tc.s && !tc.fail {
				t.Errorf("got: %s, want: %s", txt, tc.s)
			}
		})
	}
}

func TestPayload(t *testing.T) {
	tcs := []struct {
		name   string
		role   string
		input  string
		output []byte
		fail   bool
	}{
		{
			name:   "adds a message",
			role:   "user",
			input:  "foo",
			output: []byte(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"foo"}]}`),
		},
		{
			name:   "adds another message",
			role:   "admin",
			input:  "bar",
			output: []byte(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"foo"},{"role":"admin","content":"bar"}]}`),
		},
	}

	msgs := make([]message, 0, len(tcs))

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			b, err := payload(&msgs, tc.role, tc.input)
			if err != nil && !tc.fail {
				t.Errorf("unexpected error: %v", err)
			}
			if !bytes.Equal(b, tc.output) {
				t.Errorf("got: %v, want: %v", b, tc.output)
			}
		})
	}
}

type mockClient struct {
	doFn func(req *http.Request) (*http.Response, error)
}

func (m mockClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFn(req)
}

func TestRequest(t *testing.T) {
	tcs := []struct {
		name  string
		role  string
		input string
		fail  bool
		doFn  func(req *http.Request) (*http.Response, error)
	}{
		{
			name:  "handles errors",
			role:  "user",
			input: "foo",
			doFn: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("something went wrong")
			},
			fail: true,
		},
		{
			name:  "handles response",
			role:  "user",
			input: "foo",
			doFn: func(req *http.Request) (*http.Response, error) {
				r := response{
					ID:     "fake-id",
					Object: "fake-object",
					Model:  "fake-model",
					Usage: usage{
						PromptTokens:     0,
						CompletionTokens: 0,
						TotalTokens:      0,
					},
					Choices: []choices{
						{
							Message: message{
								Role:    "admin",
								Content: "This is a reply",
							},
							FinishReason: "done",
							Index:        1,
						},
					},
				}

				b, err := json.Marshal(r)
				if err != nil {
					return nil, err
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(b)),
				}, nil
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			msgs := make([]message, 0)

			b, err := payload(&msgs, tc.role, tc.input)
			if err != nil && !tc.fail {
				t.Errorf("unexpected error: %v", err)
			}

			r, err := request(&mockClient{doFn: tc.doFn}, b, os.Getenv(openAIKey))
			if err != nil && !tc.fail {
				t.Errorf("unexpected error: %v", err)
			}

			if !tc.fail && r == nil {
				t.Errorf("expected a response, got nil")
			}

		})
	}

}

func TestRun(t *testing.T) {
	tcs := []struct {
		cfg  config
		name string
		fail bool
	}{
		{
			name: "fails if OPENAI_KEY is not set",
			cfg:  config{},
			fail: true,
		},
		{
			name: "works if OPENAI_KEY is set",
			cfg: config{
				key: os.Getenv(openAIKey),
				ctx: func() context.Context {
					ctx, cancelFunc := context.WithTimeout(context.Background(), 40*time.Millisecond)
					defer cancelFunc()
					return ctx
				}(),
				client: func() httpClient {
					return mockClient{
						doFn: func(req *http.Request) (*http.Response, error) {
							r := response{
								ID:     "fake-id",
								Object: "fake-object",
								Model:  "fake-model",
								Usage: usage{
									PromptTokens:     0,
									CompletionTokens: 0,
									TotalTokens:      0,
								},
								Choices: []choices{
									{
										Message: message{
											Role:    "admin",
											Content: "This is a reply",
										},
										FinishReason: "done",
										Index:        1,
									},
								},
							}

							b, err := json.Marshal(r)
							if err != nil {
								return nil, err
							}

							return &http.Response{
								StatusCode: http.StatusOK,
								Body:       ioutil.NopCloser(bytes.NewReader(b)),
							}, nil
						},
					}
				}(),
				input: func() io.Reader {
					return strings.NewReader("foo")
				}(),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			if err := run(tc.cfg); err != nil && !tc.fail {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
