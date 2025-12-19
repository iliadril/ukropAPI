package spotify

import (
	"context"
	"fmt"

	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/zmb3/spotify/v2"
)

type SearchResult struct {
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	MusicURL     string `json:"music_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Source       string `json:"source"`
}

type Client struct {
	client *spotify.Client
}

func New(apiID, apiSecret string) (*Client, error) {
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     apiID,
		ClientSecret: apiSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create spotify client: %w", err)
	}
	httpClient := spotifyauth.New().Client(ctx, token)
	return &Client{client: spotify.New(httpClient)}, nil
}

func (s *Client) SearchMusic(query string, maxResults int) ([]SearchResult, error) {
	ctx := context.Background()
	response, err := s.client.Search(ctx, query, spotify.SearchTypeTrack, spotify.Limit(maxResults))
	if err != nil {
		return nil, fmt.Errorf("spotify search call failed: %w", err)
	}

	var results []SearchResult
	for _, track := range response.Tracks.Tracks {
		results = append(results, SearchResult{
			Artist:       track.Artists[0].Name,
			Title:        track.Name,
			MusicURL:     track.ExternalURLs["spotify"],
			ThumbnailURL: track.Album.Images[0].URL,
			Source:       "spotify",
		})
	}

	return results, nil
}
