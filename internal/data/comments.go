package data

import (
	"context"
	"database/sql"
	"time"

	"api.ukrop.pl/internal/validator"
)

type Comment struct {
	ID               int       `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	RecommendationID int       `json:"-"`
	CreatedBy        *User     `json:"created_by"`
	UserID           int       `json:"-"`
	Content          string    `json:"content"`
	Version          int       `json:"version"`
}

func ValidateComment(v *validator.Validator, comment *Comment) {
	v.Check(comment.UserID != 0, "created_by", "must be provided")

	v.Check(comment.Content != "", "content", "must be provided")
	v.Check(len(comment.Content) < 1024, "content", "must not be more than 1024 bytes long")
}

type CommentModel struct {
	DB *sql.DB
}

func (m CommentModel) Insert(comment *Comment) error {
	query := `
		INSERT INTO comments (recommendation_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, version`
	args := []any{comment.RecommendationID, comment.UserID, comment.Content}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt, &comment.Version)
}

func (m CommentModel) GetForRecommendation(recommendationID int) ([]*Comment, error) {
	query := `
		SELECT c.id, c.created_at, c.user_id, c.content, c.version,
			   u.id, u.name, u.username
		FROM comments c
		INNER JOIN users u ON u.id = c.user_id
		WHERE c.recommendation_id = $1
		ORDER BY c.created_at ASC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{recommendationID}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	comments := []*Comment{} // empty pointers

	for rows.Next() {
		var comment Comment
		comment.CreatedBy = &User{} // new user struct

		err := rows.Scan(
			&comment.ID,
			&comment.CreatedAt,
			&comment.UserID,
			&comment.Content,
			&comment.Version,
			&comment.CreatedBy.ID,
			&comment.CreatedBy.Name,
			&comment.CreatedBy.Username,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}
