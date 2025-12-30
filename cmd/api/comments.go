package main

import (
	"fmt"
	"net/http"

	"api.ukrop.pl/internal/data"
	"api.ukrop.pl/internal/validator"
)

func (app *application) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RecommendationID int    `json:"recommendation_id"`
		Content          string `json:"content"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	comment := &data.Comment{
		RecommendationID: input.RecommendationID,
		UserID:           user.ID,
		CreatedBy:        user,
		Content:          input.Content,
	}

	v := validator.New()

	if data.ValidateComment(v, comment); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Comments.Insert(comment)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"comment": comment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	app.logger.Info(fmt.Sprintf("Comment created by %s", user.Username))
}
