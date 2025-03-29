package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type mockClient struct {
	doFn func(req *http.Request) (*http.Response, error)
}

func (m mockClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFn(req)
}

type noopWriter struct{}

func (w noopWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

func TestInput(t *testing.T) {
	tcs := []struct {
		s    string
		fail bool
	}{
		{
			s: "hello",
		},
		{
			s:    "\n",
			fail: true,
		},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("TestCase%02d", i+1), func(t *testing.T) {
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
		err    error
		role   string
		input  string
		output []byte
	}{
		{
			role:   "user",
			input:  "foo",
			output: []byte(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"foo"}]}`),
		},
		{
			role:   "admin",
			input:  "bar",
			output: []byte(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"foo"},{"role":"admin","content":"bar"}]}`),
		},
	}

	msgs := make([]message, 0, len(tcs))

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("TestCase%02d", i+1), func(t *testing.T) {
			b, err := payload(&msgs, "gpt-3.5-turbo", tc.role, tc.input)
			if err != tc.err {
				t.Errorf("got: %v, want: %v", err, tc.err)
			}
			if !bytes.Equal(b, tc.output) {
				t.Errorf("got: %v, want: %v", b, tc.output)
			}
		})
	}
}

func TestRequest(t *testing.T) {
	err := errors.New("something went wrong")

	tcs := []struct {
		err   error
		doFn  func(req *http.Request) (*http.Response, error)
		role  string
		input string
	}{
		{
			role:  "user",
			input: "foo",
			doFn: func(req *http.Request) (*http.Response, error) {
				return nil, err
			},
			err: err,
		},
		{
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
					Body:       io.NopCloser(bytes.NewReader(b)),
				}, nil
			},
		},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("TestCase%02d", i+1), func(t *testing.T) {
			msgs := make([]message, 0)

			b, err := payload(&msgs, "gpt-3.5-turbo", tc.role, tc.input)
			if err != nil {
				t.Errorf("got: %v, want: %v", err, tc.err)

			}

			if _, err := request(&mockClient{doFn: tc.doFn}, b, os.Getenv(key)); err != tc.err {
				t.Errorf("got: %v, want: %v", err, tc.err)
			}

		})
	}

}

func TestRun(t *testing.T) {
	tcs := []struct {
		err error
		cfg config
	}{
		{
			cfg: config{},
			err: errKeyNotSet,
		},
		{
			cfg: config{
				key: os.Getenv(key),
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
								Body:       io.NopCloser(bytes.NewReader(b)),
							}, nil
						},
					}
				}(),
				input:  strings.NewReader(""),
				output: &noopWriter{},
			},
		},
		{
			cfg: config{
				key: os.Getenv(key),
				ctx: func() context.Context {
					ctx, cancelFunc := context.WithTimeout(context.Background(), 40*time.Millisecond)
					defer cancelFunc()
					return ctx
				}(),
				client: func() httpClient {
					return mockClient{
						doFn: func(req *http.Request) (*http.Response, error) {
							bad := `{
								"this":
							}`

							return &http.Response{
								StatusCode: http.StatusOK,
								Body:       io.NopCloser(bytes.NewReader([]byte(bad))),
							}, nil
						},
					}
				}(),
				input:  strings.NewReader(""),
				output: &noopWriter{},
			},
			err: errNoResults,
		},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("TestCase%02d", i+1), func(t *testing.T) {
			err := run(tc.cfg)
			if err != tc.err {
				t.Errorf("got: %v, want: %v", err, tc.err)
			}
		})
	}
}
