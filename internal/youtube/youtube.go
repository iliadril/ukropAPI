package youtube

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type SearchResult struct {
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	YoutubeURL   string `json:"youtube_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type YouTubeClient struct {
	service *youtube.Service
}

func New(apiKey string) (*YouTubeClient, error) {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create youtube service: %w", err)
	}

	return &YouTubeClient{service: service}, nil
}

func (y *YouTubeClient) SearchMusic(query string, maxResults int) ([]SearchResult, error) {
	call := y.service.Search.List([]string{"id", "snippet"}).
		Q(query).
		Type("video").
		VideoCategoryId("10").
		MaxResults(int64(maxResults))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("youtube search call failed: %w", err)
	}

	var results []SearchResult
	for _, item := range response.Items {
		results = append(results, SearchResult{
			Artist:       item.Snippet.ChannelTitle,
			Title:        item.Snippet.Title,
			YoutubeURL:   fmt.Sprintf("https://music.youtube.com/watch?v=%s", item.Id.VideoId),
			ThumbnailURL: item.Snippet.Thumbnails.High.Url,
		})
	}

	return results, nil
}
