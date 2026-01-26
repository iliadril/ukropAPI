package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"api.ukrop.pl/internal/validator"
)

type Reservation struct {
	ID                  int       `json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	CreatedBy           *User     `json:"created_by"`
	UserID              int       `json:"-"`
	Title               string    `json:"title"`
	Description         *string   `json:"description,omitzero"`
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time"`
	Color               *string   `json:"color,omitzero"`
	ParentReservationID int       `json:"parent_reservation_id,omitzero"`
	Version             int       `json:"version"`
}

func ValidateReservation(v *validator.Validator, reservation *Reservation) {
	v.Check(reservation.UserID != 0, "created_by", "must be provided")

	v.Check(reservation.Title != "", "title", "must be provided")
	v.Check(len(reservation.Title) <= 128, "title", "must not be more than 128 bytes long")

	v.Check(!reservation.StartTime.IsZero(), "start_time", "must be provided")
	v.Check(!reservation.EndTime.IsZero(), "end_time", "must be provided")
	v.Check(reservation.EndTime.After(reservation.StartTime), "end_time", "must be after start_time")
}

type ReservationModel struct {
	DB *sql.DB
}

func (m ReservationModel) Insert(reservation *Reservation) error {
	query := `
		INSERT INTO reservations (user_id, title, description, start_time, end_time, color, parent_reservation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, version`

	var parentID any = nil // wstawianie NULL zamiast 0-które jest nullish w golangu dla intów
	if reservation.ParentReservationID != 0 {
		parentID = reservation.ParentReservationID
	}

	args := []any{
		reservation.UserID,
		reservation.Title,
		reservation.Description,
		reservation.StartTime,
		reservation.EndTime,
		reservation.Color,
		parentID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&reservation.ID, &reservation.CreatedAt, &reservation.Version)
}

func (m ReservationModel) Get(id int) (*Reservation, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT r.id, r.created_at, r.user_id, r.title, r.description, r.start_time, r.end_time, r.color, r.parent_reservation_id, r.version,
		       u.id, u.name, u.username
		FROM reservations r
		INNER JOIN users u ON r.user_id = u.id
		WHERE r.id = $1`

	var reservation Reservation
	reservation.CreatedBy = &User{}

	var parentID sql.NullInt64
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&reservation.ID,
		&reservation.CreatedAt,
		&reservation.UserID,
		&reservation.Title,
		&reservation.Description,
		&reservation.StartTime,
		&reservation.EndTime,
		&reservation.Color,
		&parentID,
		&reservation.Version,
		&reservation.CreatedBy.ID,
		&reservation.CreatedBy.Name,
		&reservation.CreatedBy.Username,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	if parentID.Valid {
		reservation.ParentReservationID = int(parentID.Int64)
	}

	return &reservation, nil
}

func (m ReservationModel) Update(reservation *Reservation) error {
	query := `
        UPDATE reservations 
        SET title = $1, description = $2, start_time = $3, end_time = $4, color = $5, parent_reservation_id = $6, version = version + 1
        WHERE id = $7 AND version = $8
        RETURNING version`

	var parentID any = nil
	if reservation.ParentReservationID != 0 {
		parentID = reservation.ParentReservationID
	}

	args := []any{
		reservation.Title,
		reservation.Description,
		reservation.StartTime,
		reservation.EndTime,
		reservation.Color,
		parentID,
		reservation.ID,
		reservation.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&reservation.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m ReservationModel) Delete(id int) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM reservations
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m ReservationModel) GetAll(createdBy string, filters Filters) ([]*Reservation, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), r.id, r.created_at, r.user_id, r.title, r.description, r.start_time, r.end_time, r.color, r.parent_reservation_id, r.version,
		       u.id, u.name, u.username
		FROM reservations r
		INNER JOIN users u ON r.user_id = u.id
		WHERE (LOWER(u.username) = LOWER($1) OR $1 = '')
		ORDER BY %s %s, r.id DESC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{createdBy, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	reservations := []*Reservation{}

	for rows.Next() {
		var reservation Reservation
		reservation.CreatedBy = &User{}
		var parentID sql.NullInt64

		err := rows.Scan(
			&totalRecords,
			&reservation.ID,
			&reservation.CreatedAt,
			&reservation.UserID,
			&reservation.Title,
			&reservation.Description,
			&reservation.StartTime,
			&reservation.EndTime,
			&reservation.Color,
			&parentID,
			&reservation.Version,
			&reservation.CreatedBy.ID,
			&reservation.CreatedBy.Name,
			&reservation.CreatedBy.Username,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		if parentID.Valid {
			reservation.ParentReservationID = int(parentID.Int64)
		}

		reservations = append(reservations, &reservation)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return reservations, metadata, nil
}
