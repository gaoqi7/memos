package immich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	envBaseURL = "MEMOS_IMMICH_URL"
	envAPIKey  = "MEMOS_IMMICH_API_KEY"
	envAlbumName = "MEMOS_IMMICH_ALBUM_NAME"
	envAlbumID   = "MEMOS_IMMICH_ALBUM_ID"

	ReferencePrefix = "immich:"
)

var (
	uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}`)
)

type Config struct {
	BaseURL string
	APIKey  string
	AlbumName string
	AlbumID   string
}

func LoadConfig() (Config, error) {
	baseURL := strings.TrimSpace(os.Getenv(envBaseURL))
	apiKey := strings.TrimSpace(os.Getenv(envAPIKey))
	albumID := strings.TrimSpace(os.Getenv(envAlbumID))
	albumName, albumNameSet := os.LookupEnv(envAlbumName)
	albumName = strings.TrimSpace(albumName)
	if baseURL == "" || apiKey == "" {
		return Config{}, nil
	}
	if _, err := url.Parse(baseURL); err != nil {
		return Config{}, err
	}
	if !albumNameSet {
		albumName = "Memos"
	}
	return Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		AlbumName: albumName,
		AlbumID:   albumID,
	}, nil
}

func (c Config) Enabled() bool {
	return c.BaseURL != "" && c.APIKey != ""
}

func NormalizeReference(assetID string) string {
	return ReferencePrefix + assetID
}

func ParseReference(reference string) (string, bool) {
	if strings.HasPrefix(reference, ReferencePrefix) {
		assetID := strings.TrimPrefix(reference, ReferencePrefix)
		return assetID, assetID != ""
	}
	return "", false
}

func ExtractAssetIDFromLink(link string, cfg Config) (string, bool) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", false
	}

	if strings.HasPrefix(link, ReferencePrefix) {
		assetID := strings.TrimPrefix(link, ReferencePrefix)
		assetID = strings.TrimLeft(assetID, "/")
		return assetID, assetID != ""
	}
	if strings.HasPrefix(link, "immich://") {
		assetID := strings.TrimPrefix(link, "immich://")
		assetID = strings.TrimLeft(assetID, "/")
		return assetID, assetID != ""
	}

	parsed, err := url.Parse(link)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	if cfg.BaseURL != "" {
		base, err := url.Parse(cfg.BaseURL)
		if err != nil || base.Host == "" {
			return "", false
		}
		if !strings.EqualFold(parsed.Host, base.Host) {
			return "", false
		}
	}

	if match := uuidPattern.FindString(parsed.Path); match != "" {
		return match, true
	}
	for _, values := range parsed.Query() {
		for _, value := range values {
			if match := uuidPattern.FindString(value); match != "" {
				return match, true
			}
		}
	}
	return "", false
}

type Client struct {
	apiBaseURL string
	apiKey     string
	httpClient *http.Client
}

type AssetInfo struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"originalFileName"`
	OriginalMimeType string `json:"originalMimeType"`
	FileSizeInByte   int64  `json:"fileSizeInByte"`
}

type Album struct {
	ID                 string `json:"id"`
	AlbumName          string `json:"albumName"`
	Name               string `json:"name"`
	AssetCount         int    `json:"assetCount"`
	ThumbnailAssetID   string `json:"albumThumbnailAssetId"`
}

func (a Album) DisplayName() string {
	if a.AlbumName != "" {
		return a.AlbumName
	}
	return a.Name
}

type Asset struct {
	ID               string `json:"id"`
	AssetID          string `json:"assetId"`
	DeviceAssetID    string `json:"deviceAssetId"`
	UUID             string `json:"uuid"`
	OriginalFileName string `json:"originalFileName"`
	OriginalMimeType string `json:"originalMimeType"`
	FileSizeInByte   int64  `json:"fileSizeInByte"`
	Type             string `json:"type"`
}

type SearchAssetsRequest struct {
	Page     int      `json:"page"`
	Size     int      `json:"size"`
	Order    string   `json:"order,omitempty"`
	AlbumIDs []string `json:"albumIds,omitempty"`
}

type SearchAssetsResponse struct {
	Assets       []Asset
	NextPage     int
	NextPageToken string
}

type httpStatusError struct {
	StatusCode int
	Message    string
}

func (e *httpStatusError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("immich request failed: %s", e.Message)
	}
	return fmt.Sprintf("immich request failed: %d", e.StatusCode)
}

func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimSuffix(cfg.BaseURL, "/")
	if !strings.HasSuffix(base, "/api") {
		base += "/api"
	}
	_, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return &Client{
		apiBaseURL: base,
		apiKey:     cfg.APIKey,
		httpClient: http.DefaultClient,
	}, nil
}

func (c *Client) GetAssetInfo(ctx context.Context, assetID string) (*AssetInfo, error) {
	paths := []string{
		fmt.Sprintf("/assets/%s", assetID),
		fmt.Sprintf("/asset/%s", assetID),
		fmt.Sprintf("/assets/%s/info", assetID),
	}
	var lastErr error
	for _, path := range paths {
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			lastErr = fmt.Errorf("immich get asset info failed: %s", resp.Status)
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
				return nil, lastErr
			}
			continue
		}
		var info AssetInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		return &info, nil
	}
	return nil, lastErr
}

func (c *Client) ListAlbums(ctx context.Context) ([]Album, error) {
	respBody, err := c.doJSONWithFallback(ctx, http.MethodGet, []string{"/albums", "/album"}, nil)
	if err != nil {
		return nil, err
	}
	albums, err := decodeAlbums(respBody)
	if err != nil {
		return nil, err
	}
	return albums, nil
}

func (c *Client) CreateAlbum(ctx context.Context, name string) (*Album, error) {
	payload := map[string]any{
		"albumName": name,
	}
	respBody, err := c.doJSONWithFallback(ctx, http.MethodPost, []string{"/albums"}, payload)
	if err != nil {
		return nil, err
	}
	album, err := decodeAlbum(respBody)
	if err != nil {
		return nil, err
	}
	return album, nil
}

func (c *Client) AddAssetsToAlbum(ctx context.Context, albumID string, assetIDs []string) error {
	if albumID == "" || len(assetIDs) == 0 {
		return nil
	}
	assetPayload := map[string]any{"ids": assetIDs}
	paths := []string{
		fmt.Sprintf("/albums/%s/assets", albumID), // addAssetsToAlbum
	}
	var lastErr error
	for _, path := range paths {
		_, err := c.doJSONWithFallback(ctx, http.MethodPut, []string{path}, assetPayload)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryableImmichPath(err) {
			return err
		}
		_, err = c.doJSONWithFallback(ctx, http.MethodPost, []string{path}, assetPayload)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryableImmichPath(err) {
			return err
		}
	}

	return lastErr
}

func (c *Client) SearchAssets(ctx context.Context, request SearchAssetsRequest) (*SearchAssetsResponse, error) {
	paths := []string{"/search/metadata", "/search/assets", "/search"}
	respBody, err := c.doJSONWithFallback(ctx, http.MethodPost, paths, request)
	if err != nil {
		return nil, err
	}
	return decodeSearchAssets(respBody)
}

func (c *Client) ListAssets(ctx context.Context, page, size int, order string) (*SearchAssetsResponse, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 1
	}
	skip := (page - 1) * size

	queryVariants := []url.Values{
		withQuery(order, map[string]int{"page": page, "size": size}, nil),
		withQuery(order, map[string]int{"skip": skip, "take": size}, nil),
		withQuery("", map[string]int{"page": page, "size": size}, nil),
		withQuery("", map[string]int{"skip": skip, "take": size}, nil),
	}

	paths := []string{"/assets", "/asset", "/assets/owned", "/assets/all"}
	var lastErr error
	for _, path := range paths {
		for _, query := range queryVariants {
			respBody, err := c.doJSONWithQuery(ctx, http.MethodGet, path, query)
			if err == nil {
				return decodeSearchAssets(respBody)
			}
			lastErr = err
			if !isRetryableImmichPath(err) {
				return nil, err
			}
		}
	}
	return nil, lastErr
}

func (c *Client) FetchAsset(ctx context.Context, assetID, size string, download bool, requestHeaders http.Header) (*http.Response, error) {
	path := fmt.Sprintf("/assets/%s/thumbnail", assetID)
	query := url.Values{}
	if download {
		path = fmt.Sprintf("/assets/%s/original", assetID)
	} else if size != "" {
		query.Set("size", size)
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, query)
	if err != nil {
		return nil, err
	}
	if requestHeaders != nil {
		if rangeHeader := requestHeaders.Get("Range"); rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		if ifRange := requestHeaders.Get("If-Range"); ifRange != "" {
			req.Header.Set("If-Range", ifRange)
		}
	}
	return c.httpClient.Do(req)
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values) (*http.Request, error) {
	fullURL := c.apiBaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	return req, nil
}

func AddAssetToAlbum(ctx context.Context, cfg Config, assetID string) error {
	if assetID == "" {
		return nil
	}
	assetID = strings.TrimLeft(assetID, "/")
	if !uuidPattern.MatchString(assetID) {
		return nil
	}
	if cfg.BaseURL == "" || cfg.APIKey == "" {
		return nil
	}
	if cfg.AlbumID == "" && cfg.AlbumName == "" {
		return nil
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	albumID := cfg.AlbumID
	if albumID == "" {
		albums, err := client.ListAlbums(ctx)
		if err != nil {
			return err
		}
		for _, album := range albums {
			if strings.EqualFold(album.DisplayName(), cfg.AlbumName) {
				albumID = album.ID
				break
			}
		}
		if albumID == "" {
			created, err := client.CreateAlbum(ctx, cfg.AlbumName)
			if err != nil {
				return err
			}
			albumID = created.ID
		}
	}
	return client.AddAssetsToAlbum(ctx, albumID, []string{assetID})
}

func (c *Client) doJSONWithFallback(ctx context.Context, method string, paths []string, body any) ([]byte, error) {
	var lastErr error
	for _, path := range paths {
		data, err := c.doJSON(ctx, method, path, body)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !isRetryableImmichPath(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any) ([]byte, error) {
	var requestBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = strings.NewReader(string(raw))
	}
	fullURL := c.apiBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &httpStatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(bodyBytes))}
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) doJSONWithQuery(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	fullURL := c.apiBaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &httpStatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(bodyBytes))}
	}
	return io.ReadAll(resp.Body)
}

func withQuery(order string, ints map[string]int, stringsMap map[string]string) url.Values {
	query := url.Values{}
	if order != "" {
		query.Set("order", order)
	}
	for key, value := range ints {
		if value > 0 {
			query.Set(key, strconv.Itoa(value))
		}
	}
	for key, value := range stringsMap {
		if value != "" {
			query.Set(key, value)
		}
	}
	return query
}

func isRetryableImmichPath(err error) bool {
	statusErr, ok := err.(*httpStatusError)
	if !ok {
		return false
	}
	return statusErr.StatusCode == http.StatusNotFound || statusErr.StatusCode == http.StatusMethodNotAllowed
}

func decodeAlbums(data []byte) ([]Album, error) {
	var albums []Album
	if err := json.Unmarshal(data, &albums); err == nil && len(albums) > 0 {
		return albums, nil
	}
	var wrapper struct {
		Albums []Album `json:"albums"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Albums) > 0 {
		return wrapper.Albums, nil
	}
	if err := json.Unmarshal(data, &albums); err != nil {
		return nil, err
	}
	return albums, nil
}

func decodeAlbum(data []byte) (*Album, error) {
	var album Album
	if err := json.Unmarshal(data, &album); err == nil && (album.ID != "" || album.AlbumName != "" || album.Name != "") {
		return &album, nil
	}
	var wrapper struct {
		Album *Album `json:"album"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Album != nil {
		return wrapper.Album, nil
	}
	return nil, fmt.Errorf("failed to decode immich album")
}

func decodeSearchAssets(data []byte) (*SearchAssetsResponse, error) {
	assets, err := decodeAssetsFromUnknown(data)
	if err != nil {
		return nil, err
	}
	var response struct {
		NextPage      int    `json:"nextPage"`
		NextPageToken string `json:"nextPageToken"`
	}
	_ = json.Unmarshal(data, &response)
	return &SearchAssetsResponse{
		Assets:        assets,
		NextPage:      response.NextPage,
		NextPageToken: response.NextPageToken,
	}, nil
}

func decodeAssetsFromUnknown(data []byte) ([]Asset, error) {
	var rawArray []json.RawMessage
	if err := json.Unmarshal(data, &rawArray); err == nil {
		return decodeAssetSlice(rawArray)
	}

	var rawObject map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawObject); err != nil {
		return nil, err
	}

	if assetsRaw, ok := rawObject["assets"]; ok {
		if err := json.Unmarshal(assetsRaw, &rawArray); err == nil {
			return decodeAssetSlice(rawArray)
		}
		var assetsObj map[string]json.RawMessage
		if err := json.Unmarshal(assetsRaw, &assetsObj); err == nil {
			if itemsRaw, ok := assetsObj["items"]; ok {
				if err := json.Unmarshal(itemsRaw, &rawArray); err == nil {
					return decodeAssetSlice(rawArray)
				}
			}
		}
	}

	if itemsRaw, ok := rawObject["items"]; ok {
		if err := json.Unmarshal(itemsRaw, &rawArray); err == nil {
			return decodeAssetSlice(rawArray)
		}
	}

	return nil, fmt.Errorf("failed to decode immich assets")
}

func decodeAssetSlice(rawAssets []json.RawMessage) ([]Asset, error) {
	assets := make([]Asset, 0, len(rawAssets))
	for _, raw := range rawAssets {
		asset, err := decodeAssetRaw(raw)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func decodeAssetRaw(raw json.RawMessage) (Asset, error) {
	var asset Asset
	if err := json.Unmarshal(raw, &asset); err != nil {
		return Asset{}, err
	}
	if asset.ID == "" {
		asset.ID = asset.AssetID
	}
	if asset.ID == "" {
		asset.ID = asset.DeviceAssetID
	}
	if asset.ID == "" {
		asset.ID = asset.UUID
	}
	asset.ID = strings.TrimLeft(asset.ID, "/")
	if asset.ID == "" {
		if match := uuidPattern.FindString(string(raw)); match != "" {
			asset.ID = match
		}
	}
	return asset, nil
}
