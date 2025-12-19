package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"api.ukrop.pl/internal/spotify"
	"api.ukrop.pl/internal/validator"
	"api.ukrop.pl/internal/youtube"
)

type Source string

const (
	SourceSpotify Source = "spotify"
	SourceYoutube Source = "youtube"
)

type SearchResult struct {
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	MusicURL     string `json:"music_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Source       Source `json:"source"`
}

func fromYoutubeResult(r youtube.SearchResult) SearchResult {
	return SearchResult{Artist: r.Artist, Title: r.Title, MusicURL: r.MusicURL, ThumbnailURL: r.ThumbnailURL, Source: SourceYoutube}
}

func fromSpotifyResult(r spotify.SearchResult) SearchResult {
	return SearchResult{Artist: r.Artist, Title: r.Title, MusicURL: r.MusicURL, ThumbnailURL: r.ThumbnailURL, Source: SourceSpotify}
}

func (app *application) searchMusicData(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Sources []string `json:"sources"`
		Query   string   `json:"q"`
	}
	v := validator.New()
	qs := r.URL.Query()

	input.Sources = app.readCSV(qs, "sources", []string{"spotify", "youtube"})
	input.Query = app.readString(qs, "q", "")

	v.Check(len(input.Sources) <= 2, "sources", "must provide at most two sources")
	v.Check(input.Query != "", "q", "must provide a query")

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	var results []SearchResult

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ytResults, err := app.youtube.SearchMusic(ctx, input.Query, app.config.yt.maxResults)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), "youtube search call failed"):
			app.logger.Error(err.Error())
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	for _, youtubeResult := range ytResults {
		results = append(results, fromYoutubeResult(youtubeResult))
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	spResults, err := app.spotify.SearchMusic(ctx, input.Query, app.config.sp.maxResults)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), "spotify search call failed"):
			app.logger.Error(err.Error())
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	for _, spotifyResult := range spResults {
		results = append(results, fromSpotifyResult(spotifyResult))
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"tracks": results}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
