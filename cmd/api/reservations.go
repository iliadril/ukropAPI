package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"api.ukrop.pl/internal/data"
	"api.ukrop.pl/internal/validator"
)

func (app *application) createReservationHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title               string    `json:"title"`
		Description         *string   `json:"description"`
		StartTime           time.Time `json:"start_time"`
		EndTime             time.Time `json:"end_time"`
		Color               *string   `json:"color"`
		ParentReservationID int       `json:"parent_reservation_id"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	reservation := &data.Reservation{
		UserID:              user.ID,
		CreatedBy:           user,
		Title:               input.Title,
		Description:         input.Description,
		StartTime:           input.StartTime,
		EndTime:             input.EndTime,
		Color:               input.Color,
		ParentReservationID: input.ParentReservationID,
	}

	v := validator.New()

	if data.ValidateReservation(v, reservation); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Reservations.Insert(reservation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/reservations/%d", reservation.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"reservation": reservation}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	app.logger.Info(fmt.Sprintf("Reservation created by %s", user.Username))
}

func (app *application) showReservationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	reservation, err := app.models.Reservations.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reservation": reservation}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listReservationsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CreatedBy string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.CreatedBy = app.readString(qs, "created_by", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"created_at", "start_time", "-created_at", "-start_time"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	reservations, metadata, err := app.models.Reservations.GetAll(input.CreatedBy, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reservations": reservations, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
