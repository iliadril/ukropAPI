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
	SpotifyURL   string `json:"spotify_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type SpotifyClient struct {
	client *spotify.Client
}

func New(apiID, apiSecret string) (*SpotifyClient, error) {
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
	return &SpotifyClient{client: spotify.New(httpClient)}, nil
}

func (s *SpotifyClient) SearchMusic(query string) ([]SearchResult, error) {
	ctx := context.Background()
	response, err := s.client.Search(ctx, query, spotify.SearchTypeTrack)
	if err != nil {
		return nil, fmt.Errorf("spotify search call failed: %w", err)
	}

	var results []SearchResult
	for _, track := range response.Tracks.Tracks {
		results = append(results, SearchResult{
			Artist:       track.Artists[0].Name,
			Title:        track.Name,
			SpotifyURL:   track.ExternalURLs["spotify"],
			ThumbnailURL: track.Album.Images[0].URL,
		})
	}

	return results, nil
}
