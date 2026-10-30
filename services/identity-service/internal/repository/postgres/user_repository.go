package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

const (
	uniqueViolation   = "23505"
	userColumns       = "id, email, password_hash, created_at"
	selectUserByEmail = "SELECT " + userColumns + " FROM users WHERE email = $1"
	selectUserByID    = "SELECT " + userColumns + " FROM users WHERE id = $1"
)

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

func (repository *UserRepository) Create(requestContext context.Context, user domain.User) error {
	_, err := repository.pool.Exec(requestContext,
		`INSERT INTO users (`+userColumns+`) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt,
	)

	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == uniqueViolation {
		return domain.ErrUserExists
	}

	if err != nil {
		return fmt.Errorf(wrapInsertUser, err)
	}

	return nil
}

func (repository *UserRepository) FindByEmail(requestContext context.Context, email string) (domain.User, error) {
	return repository.findOne(requestContext, selectUserByEmail, email)
}

func (repository *UserRepository) FindByID(requestContext context.Context, id string) (domain.User, error) {
	return repository.findOne(requestContext, selectUserByID, id)
}

func (repository *UserRepository) findOne(requestContext context.Context, query, argument string) (domain.User, error) {
	var user domain.User

	err := repository.pool.QueryRow(requestContext, query, argument).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}

	if err != nil {
		return domain.User{}, fmt.Errorf(wrapQueryUser, err)
	}

	return user, nil
}
