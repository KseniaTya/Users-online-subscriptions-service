package subscriptions

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (Subscription, error) {
	id := uuid.New()
	row := r.db.QueryRow(ctx, `
		INSERT INTO subscriptions (id, service_name, price, user_id, start_month, end_month)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at
	`, id, input.ServiceName, input.Price, input.UserID, input.StartMonth, input.EndMonth)

	return scanSubscription(row)
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Subscription, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`, id)

	return scanSubscription(row)
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (Subscription, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE subscriptions
		SET service_name = $2,
			price = $3,
			user_id = $4,
			start_month = $5,
			end_month = $6,
			updated_at = now()
		WHERE id = $1
		RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at
	`, id, input.ServiceName, input.Price, input.UserID, input.StartMonth, input.EndMonth)

	return scanSubscription(row)
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Subscription, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
		FROM subscriptions
		WHERE ($1::uuid IS NULL OR user_id = $1)
			AND ($2 = '' OR service_name = $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, filter.UserID, filter.ServiceName, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var result []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subscriptions: %w", err)
	}

	return result, nil
}

func (r *Repository) Total(ctx context.Context, filter TotalFilter) (int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(price * (
			(EXTRACT(YEAR FROM age(LEAST(COALESCE(end_month, $2), $2), GREATEST(start_month, $1)))::int * 12)
			+ EXTRACT(MONTH FROM age(LEAST(COALESCE(end_month, $2), $2), GREATEST(start_month, $1)))::int
			+ 1
		)), 0)::bigint
		FROM subscriptions
		WHERE start_month <= $2
			AND COALESCE(end_month, $2) >= $1
			AND ($3::uuid IS NULL OR user_id = $3)
			AND ($4 = '' OR service_name = $4)
	`, filter.From, filter.To, filter.UserID, filter.ServiceName).Scan(&total); err != nil {
		return 0, fmt.Errorf("calculate total: %w", err)
	}
	return total, nil
}

type subscriptionScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(scanner subscriptionScanner) (Subscription, error) {
	var sub Subscription
	err := scanner.Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartMonth,
		&sub.EndMonth,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("scan subscription: %w", err)
	}
	return sub, nil
}
