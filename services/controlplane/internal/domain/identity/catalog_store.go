package identity

import "context"

func (s *Store) ListRoles(ctx context.Context) ([]Role, error) {
	const q = `
		SELECT code, name, description
		FROM roles
		ORDER BY code
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Role, 0)
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.Code, &r.Name, &r.Description); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ListPermissions(ctx context.Context) ([]Permission, error) {
	const q = `
		SELECT code, description
		FROM permissions
		ORDER BY code
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Permission, 0)
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Code, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}