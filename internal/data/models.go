package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Recommendations RecommendationModel
	Permissions     PermissionModel
	Tokens          TokenModel
	Users           UserModel
	Comments        CommentModel
	Reservations    ReservationModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Recommendations: RecommendationModel{DB: db},
		Permissions:     PermissionModel{DB: db},
		Tokens:          TokenModel{DB: db},
		Users:           UserModel{DB: db},
		Comments:        CommentModel{DB: db},
		Reservations:    ReservationModel{DB: db},
	}
}
