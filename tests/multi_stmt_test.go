package tests

import (
	"strings"
	"testing"

	"github.com/jespino/gopgsql/parser"
)

func TestMultiStatement(t *testing.T) {
	sql := "SELECT 1; SELECT 2; SELECT 3"
	stmts, err := parser.Parse(strings.NewReader(sql), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 3 {
		t.Errorf("expected 3 statements, got %d", len(stmts))
	}
}

// --- Complex real-world queries ---


func TestComplexQuery(t *testing.T) {
	sql := `
		WITH regional_sales AS (
			SELECT region, SUM(amount) AS total_sales
			FROM orders
			GROUP BY region
		), top_regions AS (
			SELECT region
			FROM regional_sales
			WHERE total_sales > (SELECT SUM(total_sales) / 10 FROM regional_sales)
		)
		SELECT region, product, SUM(quantity) AS product_units, SUM(amount) AS product_sales
		FROM orders
		WHERE region IN (SELECT region FROM top_regions)
		GROUP BY region, product
		ORDER BY region, product_sales DESC
		LIMIT 100
	`
	parseOK(t, sql)
}


func TestComplexInsert(t *testing.T) {
	sql := `
		INSERT INTO summary (region, total)
		SELECT region, SUM(amount)
		FROM orders
		GROUP BY region
		ON CONFLICT (region) DO UPDATE SET total = EXCLUDED.total
		RETURNING *
	`
	parseOK(t, sql)
}


func TestComplexUpdate(t *testing.T) {
	sql := `
		UPDATE accounts a
		SET balance = a.balance + t.amount
		FROM transactions t
		WHERE a.id = t.account_id AND t.processed = false
		RETURNING a.id, a.balance
	`
	parseOK(t, sql)
}

