package ai

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/usememos/memos/internal/ollama"
	"github.com/usememos/memos/internal/openai"
	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

const (
	modeGrammar = "grammar"
	modeRewrite = "rewrite"
	modeExplain = "explain"
)

type Service struct {
	store         *store.Store
	authenticator *auth.Authenticator
	client        completionClient
	provider      string
	timeout       time.Duration
}

type completionClient interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type writingAssistRequest struct {
	Content string `json:"content"`
	Mode    string `json:"mode"`
}

type writingAssistResponse struct {
	Mode   string `json:"mode"`
	Output string `json:"output"`
}

func NewService(store *store.Store, secret string, profile *profile.Profile) *Service {
	provider := strings.ToLower(strings.TrimSpace(profile.AIProvider))
	if provider == "" {
		provider = "ollama"
	}

	var (
		client  completionClient
		timeout time.Duration
	)
	switch provider {
	case "openai":
		cfg := openai.Config{
			APIKey:  profile.OpenAIAPIKey,
			BaseURL: profile.OpenAIBaseURL,
			Model:   profile.OpenAIModel,
		}
		if cfg.Enabled() {
			c, err := openai.NewClient(cfg)
			if err != nil {
				slog.Warn("failed to initialize openai client", slog.String("error", err.Error()))
			} else {
				client = c
			}
		} else {
			slog.Warn("openai provider selected but api key is empty")
		}
		timeout = time.Duration(profile.OpenAITimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 8 * time.Second
		}
	case "ollama":
		cfg := ollama.Config{
			BaseURL: profile.OllamaBaseURL,
			Model:   profile.OllamaModel,
		}
		c, err := ollama.NewClient(cfg)
		if err != nil {
			slog.Warn("failed to initialize ollama client", slog.String("error", err.Error()))
		} else {
			client = c
		}
		timeout = time.Duration(profile.OllamaTimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 120 * time.Second
		}
	default:
		slog.Warn("unsupported ai provider", slog.String("provider", provider))
		timeout = 120 * time.Second
	}

	return &Service{
		store:         store,
		authenticator: auth.NewAuthenticator(store, secret),
		client:        client,
		provider:      provider,
		timeout:       timeout,
	}
}

func (s *Service) RegisterRoutes(echoServer *echo.Echo) {
	group := echoServer.Group("/api/v1/ai")
	group.POST("/writing-assist", s.writingAssist)
}

func (s *Service) writingAssist(c echo.Context) error {
	if s.client == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "writing assistant is not configured for provider "+s.provider)
	}

	ctx := c.Request().Context()
	user, err := s.getCurrentUser(ctx, c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get current user").SetInternal(err)
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized access")
	}

	request := &writingAssistRequest{}
	if err := c.Bind(request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body").SetInternal(err)
	}

	mode := strings.TrimSpace(strings.ToLower(request.Mode))
	content := strings.TrimSpace(request.Content)
	if mode != modeGrammar && mode != modeRewrite && mode != modeExplain {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported writing assistant mode")
	}
	if content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content is required")
	}
	if len(content) > 12000 {
		return echo.NewHTTPError(http.StatusBadRequest, "content is too long")
	}

	systemPrompt, userPrompt := buildPrompts(mode, content)
	reqCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	output, err := s.client.Complete(reqCtx, systemPrompt, userPrompt)
	if err != nil {
		message := "failed to generate writing suggestion"
		if detail := strings.TrimSpace(err.Error()); detail != "" {
			message += ": " + detail
		}
		return echo.NewHTTPError(http.StatusBadGateway, message).SetInternal(err)
	}

	return c.JSON(http.StatusOK, &writingAssistResponse{
		Mode:   mode,
		Output: output,
	})
}

func buildPrompts(mode string, content string) (string, string) {
	systemPrompt := strings.Join([]string{
		"You are an English writing assistant for memo drafts.",
		"Keep markdown formatting, hashtags, links, and checklist syntax intact.",
		"Do not add greetings, disclaimers, or extra preambles.",
	}, "\n")

	switch mode {
	case modeGrammar:
		userPrompt := strings.Join([]string{
			"Correct grammar, spelling, punctuation, and awkward phrasing.",
			"Preserve the original meaning and structure.",
			"Return only the corrected memo text.",
			"",
			"Memo:",
			content,
		}, "\n")
		return systemPrompt, userPrompt
	case modeRewrite:
		userPrompt := strings.Join([]string{
			"Rewrite this memo into natural, concise, fluent English.",
			"Keep the original intent and key details.",
			"Return only the rewritten memo text.",
			"",
			"Memo:",
			content,
		}, "\n")
		return systemPrompt, userPrompt
	default:
		userPrompt := strings.Join([]string{
			"Give feedback for an English learner.",
			"Return a concise markdown bullet list with this structure:",
			"- Mistake",
			"- Why it is incorrect",
			"- Better wording",
			"Do not rewrite the full memo.",
			"",
			"Memo:",
			content,
		}, "\n")
		return systemPrompt, userPrompt
	}
}

func (s *Service) getCurrentUser(ctx context.Context, c echo.Context) (*store.User, error) {
	if authHeader := c.Request().Header.Get(echo.HeaderAuthorization); authHeader != "" {
		if user, err := s.authenticateByBearerToken(ctx, authHeader); err == nil && user != nil {
			return user, nil
		}
	}

	if cookieHeader := c.Request().Header.Get("Cookie"); cookieHeader != "" {
		if user, err := s.authenticateByRefreshToken(ctx, cookieHeader); err == nil && user != nil {
			return user, nil
		}
	}

	return nil, nil
}

func (s *Service) authenticateByBearerToken(ctx context.Context, authHeader string) (*store.User, error) {
	token := auth.ExtractBearerToken(authHeader)
	if token == "" {
		return nil, nil
	}

	if !strings.HasPrefix(token, auth.PersonalAccessTokenPrefix) {
		claims, err := s.authenticator.AuthenticateByAccessTokenV2(token)
		if err == nil && claims != nil {
			return s.store.GetUser(ctx, &store.FindUser{ID: &claims.UserID})
		}
	}

	if strings.HasPrefix(token, auth.PersonalAccessTokenPrefix) {
		user, _, err := s.authenticator.AuthenticateByPAT(ctx, token)
		if err == nil {
			return user, nil
		}
	}

	return nil, nil
}

func (s *Service) authenticateByRefreshToken(ctx context.Context, cookieHeader string) (*store.User, error) {
	refreshToken := auth.ExtractRefreshTokenFromCookie(cookieHeader)
	if refreshToken == "" {
		return nil, nil
	}

	user, _, err := s.authenticator.AuthenticateByRefreshToken(ctx, refreshToken)
	return user, err
}
