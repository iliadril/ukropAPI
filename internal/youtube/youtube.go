package youtube

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type SearchResult struct {
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	MusicURL     string `json:"music_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Source       string `json:"source"`
}

type Client struct {
	service *youtube.Service
}

func New(apiKey string) (*Client, error) {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create youtube service: %w", err)
	}

	return &Client{service: service}, nil
}

func (y *Client) SearchMusic(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	call := y.service.Search.List([]string{"id", "snippet"}).
		Q(query).
		Type("video").
		VideoCategoryId("10").
		MaxResults(int64(maxResults))

	response, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("youtube search call failed: %w", err)
	}

	var results []SearchResult
	for _, item := range response.Items {
		results = append(results, SearchResult{
			Artist:       item.Snippet.ChannelTitle,
			Title:        item.Snippet.Title,
			MusicURL:     fmt.Sprintf("https://music.youtube.com/watch?v=%s", item.Id.VideoId),
			ThumbnailURL: item.Snippet.Thumbnails.High.Url,
			Source:       "youtube",
		})
	}

	return results, nil
}
