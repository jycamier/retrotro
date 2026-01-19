package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

var ErrNotFound = errors.New("record not found")

// UserRepository handles user database operations
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oidc_subject, oidc_issuer,
		       is_admin, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsAdmin,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

// FindByOIDC finds a user by OIDC subject and issuer
func (r *UserRepository) FindByOIDC(ctx context.Context, subject, issuer string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oidc_subject, oidc_issuer,
		       is_admin, last_login_at, created_at, updated_at
		FROM users WHERE oidc_subject = $1 AND oidc_issuer = $2
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, subject, issuer).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsAdmin,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oidc_subject, oidc_issuer,
		       is_admin, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL,
		&user.OIDCSubject, &user.OIDCIssuer, &user.IsAdmin,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
		INSERT INTO users (id, email, display_name, avatar_url, oidc_subject, oidc_issuer, is_admin)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.DisplayName, user.AvatarURL,
		user.OIDCSubject, user.OIDCIssuer, user.IsAdmin,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $2, display_name = $3, avatar_url = $4, is_admin = $5, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.DisplayName, user.AvatarURL, user.IsAdmin,
	)

	return err
}

// UpdateOIDC updates a user's OIDC fields and other attributes
func (r *UserRepository) UpdateOIDC(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET oidc_subject = $2, oidc_issuer = $3, display_name = $4, is_admin = $5, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.OIDCSubject, user.OIDCIssuer, user.DisplayName, user.IsAdmin,
	)

	return err
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

// ListAll returns all users
func (r *UserRepository) ListAll(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oidc_subject, oidc_issuer,
		       is_admin, last_login_at, created_at, updated_at
		FROM users
		ORDER BY display_name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL,
			&user.OIDCSubject, &user.OIDCIssuer, &user.IsAdmin,
			&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if users == nil {
		users = []*models.User{}
	}

	return users, nil
}

// FindOrCreate finds a user by OIDC or creates a new one
func (r *UserRepository) FindOrCreate(ctx context.Context, subject, issuer, email, name string, avatarURL *string) (*models.User, bool, error) {
	user, err := r.FindByOIDC(ctx, subject, issuer)
	if err == nil {
		// Update user info if changed
		if user.Email != email || user.DisplayName != name {
			user.Email = email
			user.DisplayName = name
			user.AvatarURL = avatarURL
			_ = r.Update(ctx, user)
		}
		return user, false, nil
	}

	if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}

	// Create new user
	user = &models.User{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: name,
		AvatarURL:   avatarURL,
		OIDCSubject: subject,
		OIDCIssuer:  issuer,
		IsAdmin:     false,
	}

	user, err = r.Create(ctx, user)
	if err != nil {
		return nil, false, err
	}

	return user, true, nil
}
