package immich

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	immichclient "github.com/usememos/memos/internal/immich"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

type Service struct {
	store         *store.Store
	authenticator *auth.Authenticator
}

func NewService(store *store.Store, secret string) *Service {
	return &Service{
		store:         store,
		authenticator: auth.NewAuthenticator(store, secret),
	}
}

func (s *Service) RegisterRoutes(echoServer *echo.Echo) {
	group := echoServer.Group("/api/immich")
	group.GET("/albums", s.listAlbums)
	group.GET("/assets", s.listAssets)
}

func (s *Service) listAlbums(c echo.Context) error {
	ctx := c.Request().Context()
	user, err := s.getCurrentUser(ctx, c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get current user").SetInternal(err)
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized access")
	}

	cfg, err := immichclient.LoadConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load immich config").SetInternal(err)
	}
	if !cfg.Enabled() {
		return echo.NewHTTPError(http.StatusBadRequest, "immich is not configured")
	}

	client, err := immichclient.NewClient(cfg)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to initialize immich client").SetInternal(err)
	}

	albums, err := client.ListAlbums(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "failed to fetch immich albums").SetInternal(err)
	}

	response := make([]map[string]any, 0, len(albums))
	for _, album := range albums {
		response = append(response, map[string]any{
			"id":               album.ID,
			"name":             album.DisplayName(),
			"assetCount":       album.AssetCount,
			"thumbnailAssetId": album.ThumbnailAssetID,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{"albums": response})
}

func (s *Service) listAssets(c echo.Context) error {
	ctx := c.Request().Context()
	user, err := s.getCurrentUser(ctx, c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get current user").SetInternal(err)
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized access")
	}

	cfg, err := immichclient.LoadConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load immich config").SetInternal(err)
	}
	if !cfg.Enabled() {
		return echo.NewHTTPError(http.StatusBadRequest, "immich is not configured")
	}

	pageSize := 60
	if raw := c.QueryParam("pageSize"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	page := 1
	if raw := c.QueryParam("pageToken"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}

	client, err := immichclient.NewClient(cfg)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to initialize immich client").SetInternal(err)
	}

	searchResponse, err := client.ListAssets(ctx, page, pageSize, "desc")
	if err != nil {
		searchResponse, err = client.SearchAssets(ctx, immichclient.SearchAssetsRequest{
			Page:  page,
			Size:  pageSize,
			Order: "desc",
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, "failed to fetch immich assets").SetInternal(err)
		}
	}

	assets := make([]map[string]any, 0, len(searchResponse.Assets))
	for _, asset := range searchResponse.Assets {
		assets = append(assets, map[string]any{
			"id":           asset.ID,
			"filename":     asset.OriginalFileName,
			"mimeType":     asset.OriginalMimeType,
			"size":         asset.FileSizeInByte,
			"type":         asset.Type,
			"thumbnailUrl": "/file/immich/" + asset.ID + "?size=thumbnail",
			"previewUrl":   "/file/immich/" + asset.ID + "?size=fullsize",
		})
	}

	nextPageToken := searchResponse.NextPageToken
	if nextPageToken == "" && searchResponse.NextPage > 0 {
		nextPageToken = strconv.Itoa(searchResponse.NextPage)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"assets":        assets,
		"nextPageToken": nextPageToken,
	})
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
