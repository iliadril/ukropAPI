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
	ID          int        `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   *User      `json:"created_by"`
	UserID      int        `json:"-"`
	Artist      string     `json:"artist"`
	Title       string     `json:"title"`
	CoverURL    string     `json:"cover_url,omitzero"`
	YTLink      string     `json:"yt_link,omitzero"`
	SpotifyLink string     `json:"spotify_link,omitzero"`
	Comment     string     `json:"comment,omitzero"`
	IsPublic    bool       `json:"is_public"`
	Version     int        `json:"version"`
	Comments    []*Comment `json:"comments,omitzero"` // TODO move this away from here
}

func ValidateRecommendation(v *validator.Validator, recommendation *Recommendation) {
	v.Check(recommendation.Artist != "", "artist", "must be provided")
	v.Check(len(recommendation.Artist) <= 128, "artist", "must not be more than 128 bytes long")

	v.Check(recommendation.Title != "", "title", "must be provided")
	v.Check(len(recommendation.Title) <= 128, "title", "must not be more than 128 bytes long")

	v.Check(recommendation.UserID != 0, "created_by", "must be provided")

	v.Check(recommendation.YTLink != "" || recommendation.SpotifyLink != "", "yt_link|spotify_link", "must be provided")
}

type RecommendationModel struct {
	DB *sql.DB
}

func (m RecommendationModel) Insert(recommendation *Recommendation) error {
	query := `
		INSERT INTO recommendations (user_id, artist, title, cover_url, yt_link, spotify_link, comment, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, version`
	args := []any{
		recommendation.UserID, recommendation.Artist, recommendation.Title, recommendation.CoverURL,
		recommendation.YTLink, recommendation.SpotifyLink, recommendation.Comment, recommendation.IsPublic,
	} // TODO this inserts empty strings "" rather than nulls. Fix it

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&recommendation.ID, &recommendation.CreatedAt, &recommendation.Version)
}

func (m RecommendationModel) Get(id int) (*Recommendation, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT r.id, r.created_at, r.user_id, r.artist, r.title, r.cover_url, r.yt_link, r.spotify_link, r.comment, r.is_public, r.version,
		       u.id, u.name, u.username
		FROM recommendations r
		INNER JOIN users u ON r.user_id = u.id
		WHERE r.id = $1`

	var recommendation Recommendation
	recommendation.CreatedBy = &User{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&recommendation.ID,
		&recommendation.CreatedAt,
		&recommendation.UserID,
		&recommendation.Artist,
		&recommendation.Title,
		&recommendation.CoverURL,
		&recommendation.YTLink,
		&recommendation.SpotifyLink,
		&recommendation.Comment,
		&recommendation.IsPublic,
		&recommendation.Version,
		&recommendation.CreatedBy.ID,
		&recommendation.CreatedBy.Name,
		&recommendation.CreatedBy.Username)

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
        SET artist = $1, title = $2, cover_url = $3, yt_link = $4, spotify_link = $5, comment = $6, is_public = $7, version = version + 1
        WHERE id = $8 AND version = $9
        RETURNING version`

	args := []any{
		recommendation.Artist,
		recommendation.Title,
		recommendation.CoverURL,
		recommendation.YTLink,
		recommendation.SpotifyLink,
		recommendation.Comment,
		recommendation.IsPublic,
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

func (m RecommendationModel) GetAll(createdAt time.Time, createdBy, title string, privatePermissions bool, filters Filters) ([]*Recommendation, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), r.id, r.created_at, r.user_id, r.artist, r.title, r.cover_url, r.yt_link, r.spotify_link, r.comment, r.is_public, r.version,
		       u.id, u.name, u.username
		FROM recommendations r
		INNER JOIN users u ON r.user_id = u.id
		WHERE (r.created_at::date = $1 OR $1 = '0001-01-01'::date)
		AND (LOWER(u.username) = LOWER($2) OR $2 = '')
		AND (to_tsvector('simple', r.title) @@ plainto_tsquery('simple', $3) OR $3 = '')
		AND ($4 = true OR r.is_public = true)
		ORDER BY %s %s, r.id DESC
		LIMIT $5 OFFSET $6`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{createdAt, createdBy, title, privatePermissions, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close() // important to close the resultset before GetAll() returns

	totalRecords := 0
	recommendations := []*Recommendation{} // empty slice and not a literal because json v1 is stupid

	for rows.Next() {
		var recommendation Recommendation
		recommendation.CreatedBy = &User{} // Initialize User struct

		err := rows.Scan(
			&totalRecords,
			&recommendation.ID,
			&recommendation.CreatedAt,
			&recommendation.UserID,
			&recommendation.Artist,
			&recommendation.Title,
			&recommendation.CoverURL,
			&recommendation.YTLink,
			&recommendation.SpotifyLink,
			&recommendation.Comment,
			&recommendation.IsPublic,
			&recommendation.Version,
			&recommendation.CreatedBy.ID,
			&recommendation.CreatedBy.Name,
			&recommendation.CreatedBy.Username,
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
