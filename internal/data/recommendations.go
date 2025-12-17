package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"api.ukrop.pl/internal/validator"
	_ "github.com/lib/pq"
)

type Recommendation struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	Title       string    `json:"title"`
	YTLink      string    `json:"yt_link,omitzero"`
	SpotifyLink string    `json:"spotify_link,omitzero"`
	Comment     string    `json:"comment,omitzero"`
	Version     int       `json:"version"`
}

func ValidateRecommendation(v *validator.Validator, recommendation *Recommendation) {
	v.Check(recommendation.Title != "", "title", "must be provided")
	v.Check(len(recommendation.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(recommendation.CreatedBy != "", "created_by", "must be provided")
	v.Check(len(recommendation.CreatedBy) <= 128, "title", "must not be more than 128 bytes long")

	v.Check(recommendation.YTLink != "" || recommendation.SpotifyLink != "", "yt_link|spotify_link", "must be provided")
}

type RecommendationModel struct {
	DB *sql.DB
}

func (m RecommendationModel) Insert(recommendation *Recommendation) error {
	query := `
		INSERT INTO recommendations (created_by, title, yt_link, spotify_link, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version`
	args := []any{recommendation.CreatedBy, recommendation.Title, recommendation.YTLink, recommendation.SpotifyLink, recommendation.Comment}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&recommendation.ID, &recommendation.CreatedAt, &recommendation.Version)
}

func (m RecommendationModel) Get(id int) (*Recommendation, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, created_at, created_by, title, yt_link, spotify_link, comment, version
		FROM recommendations
		WHERE id = $1`

	var recommendation Recommendation

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&recommendation.ID,
		&recommendation.CreatedAt,
		&recommendation.CreatedBy,
		&recommendation.Title,
		&recommendation.YTLink,
		&recommendation.SpotifyLink,
		&recommendation.Comment,
		&recommendation.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &recommendation, nil
}

func (m RecommendationModel) Update(recommendation *Recommendation) error {
	query := `
        UPDATE recommendations 
        SET created_by = $1, title = $2, yt_link = $3, spotify_link = $4, comment = $5, version = version + 1
        WHERE id = $6 AND version = $7
        RETURNING version`

	args := []any{
		recommendation.CreatedBy,
		recommendation.Title,
		recommendation.YTLink,
		recommendation.SpotifyLink,
		recommendation.Comment,
		recommendation.ID,
		recommendation.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&recommendation.Version)
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

func (m RecommendationModel) Delete(id int) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM recommendations
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

func (m RecommendationModel) GetAll(createdAt time.Time, createdBy, title string, filters Filters) ([]*Recommendation, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, created_by, title, yt_link, spotify_link, comment, version
		FROM recommendations
		WHERE (created_at::date = $1 OR $1 = '0001-01-01'::date)
		AND (LOWER(created_by) = LOWER($2) OR $2 = '')
		AND (to_tsvector('simple', title) @@ plainto_tsquery('simple', $3) OR $3 = '')
		ORDER BY %s %s, id DESC
		LIMIT $4 OFFSET $5`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{createdAt, createdBy, title, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close() // important to close the resultset before GetAll() returns

	totalRecords := 0
	recommendations := []*Recommendation{} // empty slice and not a literal because json v1 is stupid

	for rows.Next() {
		var recommendation Recommendation
		err := rows.Scan(
			&totalRecords,
			&recommendation.ID,
			&recommendation.CreatedAt,
			&recommendation.CreatedBy,
			&recommendation.Title,
			&recommendation.YTLink,
			&recommendation.SpotifyLink,
			&recommendation.Comment,
			&recommendation.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		recommendations = append(recommendations, &recommendation)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return recommendations, metadata, nil
}
