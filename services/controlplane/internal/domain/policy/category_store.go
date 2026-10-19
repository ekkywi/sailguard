package policy

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Category struct {
	Code        string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CategorySignature struct {
	ID           uuid.UUID
	CategoryCode string
	RuleType     string
	Operator     string
	Value        string
	CreatedAt    time.Time
}

type CreateCategoryInput struct {
	Code        string
	Name        string
	Description string
}

type UpdateCategoryInput struct {
	Name        string
	Description string
}

type AddSignatureInput struct {
	RuleType string
	Operator string
	Value    string
}

func normalizeCategoryCode(code string) (string, error) {
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" || len(code) > 64 {
		return "", ErrInvalid
	}
	for _, r := range code {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return "", ErrInvalid
	}
	return code, nil
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	const q = `
		SELECT code, name, description, created_at, updated_at
		FROM categories
		ORDER BY name
	`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.Code, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) FindCategoryByCode(ctx context.Context, code string) (*Category, error) {
	code, err := normalizeCategoryCode(code)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT code, name, description, created_at, updated_at
		FROM categories
		WHERE code = $1
	`
	var c Category
	err = s.pool.QueryRow(ctx, q, code).Scan(
		&c.Code, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateCategory(ctx context.Context, in CreateCategoryInput) (*Category, error) {
	code, err := normalizeCategoryCode(in.Code)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalid
	}
	desc := strings.TrimSpace(in.Description)

	const q = `
		INSERT INTO categories (code, name, description)
		VALUES ($1, $2, $3)
		RETURNING code, name, description, created_at, updated_at
	`
	var c Category
	err = s.pool.QueryRow(ctx, q, code, name, desc).Scan(
		&c.Code, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpdateCategory(ctx context.Context, code string, in UpdateCategoryInput) (*Category, error) {
	code, err := normalizeCategoryCode(code)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalid
	}
	desc := strings.TrimSpace(in.Description)

	const q = `
		UPDATE categories
		SET name = $2,
		    description = $3,
		    updated_at = now()
		WHERE code = $1
		RETURNING code, name, description, created_at, updated_at
	`
	var c Category
	err = s.pool.QueryRow(ctx, q, code, name, desc).Scan(
		&c.Code, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) DeleteCategory(ctx context.Context, code string) error {
	code, err := normalizeCategoryCode(code)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM categories WHERE code = $1`, code)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListSignaturesByCategory(ctx context.Context, code string) ([]CategorySignature, error) {
	code, err := normalizeCategoryCode(code)
	if err != nil {
		return nil, err
	}
	if _, err := s.FindCategoryByCode(ctx, code); err != nil {
		return nil, err
	}

	const q = `
		SELECT id, category_code, rule_type, operator, value, created_at
		FROM category_signatures
		WHERE category_code = $1
		ORDER BY created_at, value
	`
	rows, err := s.pool.Query(ctx, q, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CategorySignature, 0)
	for rows.Next() {
		var sig CategorySignature
		if err := rows.Scan(
			&sig.ID, &sig.CategoryCode, &sig.RuleType, &sig.Operator, &sig.Value, &sig.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, sig)
	}
	return out, rows.Err()
}

func (s *Store) AddSignature(ctx context.Context, code string, in AddSignatureInput) (*CategorySignature, error) {
	code, err := normalizeCategoryCode(code)
	if err != nil {
		return nil, err
	}
	if _, err := s.FindCategoryByCode(ctx, code); err != nil {
		return nil, err
	}

	ruleType := strings.TrimSpace(strings.ToLower(in.RuleType))
	op := strings.TrimSpace(strings.ToLower(in.Operator))
	if op == "" {
		op = "equals"
	}
	value := strings.TrimSpace(in.Value)
	if value == "" {
		return nil, ErrInvalid
	}
	switch ruleType {
	case "process_name", "path", "hash_sha256", "publisher", "product_name":
	default:
		return nil, ErrInvalid
	}
	switch op {
	case "equals", "prefix", "contains":
	default:
		return nil, ErrInvalid
	}

	const q = `
		INSERT INTO category_signatures (category_code, rule_type, operator, value)
		VALUES ($1, $2, $3, $4)
		RETURNING id, category_code, rule_type, operator, value, created_at
	`
	var sig CategorySignature
	err = s.pool.QueryRow(ctx, q, code, ruleType, op, value).Scan(
		&sig.ID, &sig.CategoryCode, &sig.RuleType, &sig.Operator, &sig.Value, &sig.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sig, nil
}

func (s *Store) DeleteSignature(ctx context.Context, code string, signatureID uuid.UUID) error {
	code, err := normalizeCategoryCode(code)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM category_signatures
		WHERE id = $1 AND category_code = $2
	`, signatureID, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
