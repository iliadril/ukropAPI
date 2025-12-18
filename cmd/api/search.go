package main

import (
	"net/http"
	"strings"
)

func (app *application) searchYoutubeHandler(w http.ResponseWriter, r *http.Request) {
	query, err := app.readQueryParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
	}

	suggestedLinks, err := app.youtube.SearchMusic(query, app.config.yt.maxResults)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), "youtube search call failed"):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"results": suggestedLinks}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) searchSpotifyHandler(w http.ResponseWriter, r *http.Request) {
	query, err := app.readQueryParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
	}

	suggestedLinks, err := app.spotify.SearchMusic(query)
	if err != nil {
		switch {
		case strings.HasPrefix(err.Error(), "spotify search call failed"):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"results": suggestedLinks}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
