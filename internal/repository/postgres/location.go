package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fleet-management-poc/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type LocationRepository struct{ db *sql.DB }

func NewLocationRepository(db *sql.DB) *LocationRepository {
	return &LocationRepository{
		db: db,
	}
}

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = sql.Open("pgx", dsn)
		if err == nil {
			err = db.PingContext(ctx)
			if err == nil {
				return db, nil
			}
			_ = db.Close()
		}
		time.Sleep(time.Second)
	}
	return nil, err
}
func (r *LocationRepository) Save(ctx context.Context, l domain.Location) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vehicle_loctions 
		    ( vehicle_id, latitude, longitude, "timestamp" ) 
			VALUES 
			( $1, $2, $3, $4 )`,
		l.VehicleID, l.Latitude, l.Longitude, l.Timestamp)
	return err
}
func (r *LocationRepository) Latest(ctx context.Context, id string) (domain.Location, error) {
	var l domain.Location
	err := r.db.QueryRowContext(ctx, `
		SELECT 
		    vehicle_id,latitude,longitude,"timestamp" 
		FROM vehicle_loctions 
		WHERE vehicle_id = $1
		ORDER BY "timestamp" DESC, id DESC 
		LIMIT 1`, id,
	).Scan(&l.VehicleID, &l.Latitude, &l.Longitude, &l.Timestamp)

	if errors.Is(err, sql.ErrNoRows) {
		return l, sql.ErrNoRows
	}
	return l, err
}
func (r *LocationRepository) History(ctx context.Context, id string, start, end int64) ([]domain.Location, error) {
	rows, err := r.db.QueryContext(ctx, `
	SELECT 
	    vehicle_id,latitude,longitude,"timestamp"
	FROM vehicle_loctions 
	WHERE vehicle_id = $1 
	  AND "timestamp" >= $2 AND "timestamp" <= $3 
	ORDER BY "timestamp" ASC,id ASC`,
		id, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Location, 0)
	for rows.Next() {
		var l domain.Location
		if err := rows.Scan(&l.VehicleID, &l.Latitude, &l.Longitude, &l.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
