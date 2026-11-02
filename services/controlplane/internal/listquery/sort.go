package listquery

import "strings"

// OrderBy returns a safe ORDER BY expression from allowlisted columns.
// sortKey is a client-facing key; allowed maps key → SQL expression (no user input).
// defaultExpr is used when sortKey is empty or unknown (include direction).
func OrderBy(sortKey, order string, allowed map[string]string, defaultExpr string) string {
	key := strings.ToLower(strings.TrimSpace(sortKey))
	col, ok := allowed[key]
	if !ok || col == "" {
		return defaultExpr
	}
	dir := "ASC"
	if strings.EqualFold(strings.TrimSpace(order), "desc") {
		dir = "DESC"
	}
	return col + " " + dir
}
