package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultBaseURL = "http://127.0.0.1:11434"
	defaultModel   = "qwen3:8b"
)

type Config struct {
	BaseURL string
	Model   string
}

type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.Wrap(err, "invalid ollama base url")
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid ollama base url")
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}

	return &Client{
		baseURL:    baseURL,
		model:      model,
		httpClient: http.DefaultClient,
	}, nil
}

type generateRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	System      string   `json:"system,omitempty"`
	Stream      bool     `json:"stream"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type generateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	temperature := 0.2
	requestBody, err := json.Marshal(generateRequest{
		Model:       c.model,
		System:      systemPrompt,
		Prompt:      userPrompt,
		Stream:      false,
		Temperature: &temperature,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal ollama request")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(requestBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to create ollama request")
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", errors.Wrap(err, "failed to call ollama api")
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read ollama response")
	}

	parsed := &generateResponse{}
	if err := json.Unmarshal(body, parsed); err != nil {
		return "", errors.Wrap(err, "failed to parse ollama response")
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if detail := strings.TrimSpace(parsed.Error); detail != "" {
			return "", errors.New(detail)
		}
		return "", errors.New("ollama request failed")
	}

	if detail := strings.TrimSpace(parsed.Error); detail != "" {
		return "", errors.New(detail)
	}

	content := strings.TrimSpace(parsed.Response)
	if content == "" {
		return "", errors.New("ollama returned empty content")
	}
	return content, nil
}
