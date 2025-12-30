package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"api.ukrop.pl/internal/data"
	"api.ukrop.pl/internal/validator"
)

func (app *application) createRecommendationHandler(w http.ResponseWriter, r *http.Request) {
	var input struct { // sort of an input DTO
		Artist      string `json:"artist"`
		Title       string `json:"title"`
		CoverURL    string `json:"cover_url"`
		YTLink      string `json:"yt_link"`
		SpotifyLink string `json:"spotify_link"`
		Comment     string `json:"comment"`
		IsPublic    bool   `json:"is_public"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	recommendation := &data.Recommendation{
		UserID:      user.ID,
		CreatedBy:   user,
		Artist:      input.Artist,
		Title:       input.Title,
		CoverURL:    input.CoverURL,
		YTLink:      input.YTLink,
		SpotifyLink: input.SpotifyLink,
		Comment:     input.Comment,
		IsPublic:    input.IsPublic,
	}

	v := validator.New()

	if data.ValidateRecommendation(v, recommendation); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Recommendations.Insert(recommendation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header) // add location header for the user
	headers.Set("Location", fmt.Sprintf("/v1/recommendations/%d", recommendation.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"recommendation": recommendation}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	app.logger.Info(fmt.Sprintf("Recommendation created by %s", user.Username))
}

func (app *application) showRecommendationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	recommendation, err := app.models.Recommendations.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	comments, err := app.models.Comments.GetForRecommendation(recommendation.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	recommendation.Comments = comments

	err = app.writeJSON(w, http.StatusOK, envelope{"recommendation": recommendation}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateRecommendationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	recommendation, err := app.models.Recommendations.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Title       *string `json:"title"`
		YTLink      *string `json:"yt_link"`
		SpotifyLink *string `json:"spotify_link"`
		Comment     *string `json:"comment"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		recommendation.Title = *input.Title // dereference the pointer to get the value
	}
	if input.YTLink != nil {
		recommendation.YTLink = *input.YTLink
	}
	if input.SpotifyLink != nil {
		recommendation.SpotifyLink = *input.SpotifyLink
	}
	if input.Comment != nil {
		recommendation.Comment = *input.Comment
	}

	v := validator.New()
	if data.ValidateRecommendation(v, recommendation); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Recommendations.Update(recommendation)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"recommendation": recommendation}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteRecommendationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Recommendations.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "recommend successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listRecommendationsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CreatedAt time.Time
		CreatedBy string
		Title     string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.CreatedAt = app.readDate(qs, "created_at", time.Time{}, v)
	input.CreatedBy = app.readString(qs, "created_by", "")
	input.Title = app.readString(qs, "title", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"created_at", "created_by", "-created_at", "-created_by"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// check for public permissions
	user := app.contextGetUser(r)
	permissions, err := app.models.Permissions.GetAllForUser(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	privatePermissions := permissions.Include("recommendations:write")

	recommendations, metadata, err := app.models.Recommendations.GetAll(input.CreatedAt, input.CreatedBy, input.Title, privatePermissions, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"recommendations": recommendations, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
