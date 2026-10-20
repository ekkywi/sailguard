package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	const q = `
		SELECT id, email, name, password_hash, auth_provider, is_active,
				last_login_at, created_at, updated_at
		FROM users
		ORDER BY email
	`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.Name,
			&u.PasswordHash,
			&u.AuthProvider,
			&u.IsActive,
			&u.LastLoginAt,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func (s *Store) CreateUser(
	ctx context.Context,
	email, name, passwordHash string,
	roleCodes []string,
) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if email == "" || name == "" || passwordHash == "" {
		return nil, errors.New("email, name, and password are required")
	}
	if len(roleCodes) == 0 {
		return nil, ErrInvalidRole
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const insertUser = `
		INSERT INTO users (email, name, password_hash, auth_provider)
		VALUES ($1, $2, $3, 'local')
		RETURNING id, email, name, password_hash, auth_provider, is_active,
					last_login_at, created_at, updated_at
	`
	var u User
	err = tx.QueryRow(ctx, insertUser, email, name, passwordHash).Scan(
		&u.ID,
		&u.Email,
		&u.Name,
		&u.PasswordHash,
		&u.AuthProvider,
		&u.IsActive,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, err
	}

	for _, code := range roleCodes {
		code = strings.TrimSpace(code)
		var roleID string
		err := tx.QueryRow(ctx,
			`SELECT id::text FROM roles WHERE code = $1`, code,
		).Scan(&roleID)
		if err != nil {
			return nil, ErrInvalidRole
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
			u.ID, roleID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	const q = `
		SELECT COUNT(*)::bigint
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE u.is_active = true
		  AND r.code = 'super_admin'
	`
	var n int64
	err := s.pool.QueryRow(ctx, q).Scan(&n)
	return n, err
}

func (s *Store) SetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`,
		id, passwordHash,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) replaceUserRolesTx(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	roleCodes []string,
) error {
	if len(roleCodes) == 0 {
		return ErrInvalidRole
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, code := range roleCodes {
		code = strings.TrimSpace(code)
		var roleID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE code = $1`, code).Scan(&roleID)
		if err != nil {
			return ErrInvalidRole
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) VALUES ($1,  $2)`,
			userID, roleID,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	isActive *bool,
	roleCodes *[]string,
) (*User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM users WHERE id = $1`, id).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return nil, errors.New("name cannot be empty")
		}
		if _, err := tx.Exec(ctx,
			`UPDATE users SET name = $2, updated_at = now() WHERE id = $1`,
			id, n,
		); err != nil {
			return nil, err
		}
	}

	if isActive != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE users SET is_active = $2, updated_at = now() WHERE id = $1`,
			id, *isActive,
		); err != nil {
			return nil, err
		}
	}

	if roleCodes != nil {
		if err := s.replaceUserRolesTx(ctx, tx, id, *roleCodes); err != nil {
			return nil, err
		}
	}

	const q = `
		SELECT id, email, name, password_hash, auth_provider, is_active,
		       last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	var u User
	if err := tx.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider, &u.IsActive,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u, nil
}