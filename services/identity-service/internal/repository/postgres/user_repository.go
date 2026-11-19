package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

const upsertUser = `
	INSERT INTO users (id, email, name, created_at, updated_at)
	VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, $4)
	ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
	RETURNING id, COALESCE(email, ''), COALESCE(name, ''), created_at, updated_at`

//go:embed schema.sql
var schema string

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (repository *UserRepository) Migrate(requestContext context.Context) error {
	if _, err := repository.pool.Exec(requestContext, schema); err != nil {
		return fmt.Errorf(wrapApplyIdentitySchema, err)
	}

	return nil
}

func (repository *UserRepository) Upsert(requestContext context.Context, user domain.User) (domain.User, error) {
	var saved domain.User

	err := repository.pool.QueryRow(requestContext, upsertUser, user.ID, user.Email, user.Name, user.UpdatedAt).
		Scan(&saved.ID, &saved.Email, &saved.Name, &saved.CreatedAt, &saved.UpdatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf(wrapUpsertUser, err)
	}

	return saved, nil
}
