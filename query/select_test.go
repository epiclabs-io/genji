package query_test

import (
	"bytes"
	"database/sql"
	"testing"

	"github.com/asdine/genji"
	"github.com/asdine/genji/document"
	"github.com/asdine/genji/engine/memoryengine"
	"github.com/stretchr/testify/require"
)

func TestSelectStmt(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		fails    bool
		expected string
		params   []interface{}
	}{
		{"No cond", "SELECT * FROM test", false, "foo1,bar1,baz1,1\nfoo2,bar1,1,2\nfoo3,bar2,3\n", nil},
		{"Multiple wildcards cond", "SELECT *, *, a FROM test", false, "foo1,bar1,baz1,1,foo1,bar1,baz1,1,foo1\nfoo2,bar1,1,2,foo2,bar1,1,2,foo2\nfoo3,bar2,3,foo3,bar2,3\n", nil},
		{"With fields", "SELECT a, c FROM test", false, "foo1,baz1\nfoo2\n\n", nil},
		{"With eq cond", "SELECT * FROM test WHERE b = 'bar1'", false, "foo1,bar1,baz1,1\nfoo2,bar1,1,2\n", nil},
		{"With neq cond", "SELECT * FROM test WHERE a != 'foo1'", false, "foo2,bar1,1,2\nfoo3,bar2,3\n", nil},
		{"With gt cond", "SELECT * FROM test WHERE b > 'bar1'", false, "", nil},
		{"With lt cond", "SELECT * FROM test WHERE a < 'zzzzz'", false, "foo1,bar1,baz1,1\nfoo2,bar1,1,2\n", nil},
		{"With lte cond", "SELECT * FROM test WHERE a <= 'foo3'", false, "foo1,bar1,baz1,1\nfoo2,bar1,1,2\n", nil},
		{"With field comparison", "SELECT * FROM test WHERE b < a", false, "foo1,bar1,baz1,1\nfoo2,bar1,1,2\n", nil},
		{"With limit", "SELECT * FROM test WHERE b = 'bar1' LIMIT 1", false, "foo1,bar1,baz1,1\n", nil},
		{"With offset", "SELECT *, key() FROM test WHERE b = 'bar1' OFFSET 1", false, "foo2,bar1,1,2,2\n", nil},
		{"With limit then offset", "SELECT * FROM test WHERE b = 'bar1' LIMIT 1 OFFSET 1", false, "foo2,bar1,1,2\n", nil},
		{"With offset then limit", "SELECT * FROM test WHERE b = 'bar1' OFFSET 1 LIMIT 1", true, "", nil},
		{"With positional params", "SELECT * FROM test WHERE a = ? OR d = ?", false, "foo1,bar1,baz1,1\nfoo3,bar2,3\n", []interface{}{"foo1", "foo3"}},
		{"With named params", "SELECT * FROM test WHERE a = $a OR d = $d", false, "foo1,bar1,baz1,1\nfoo3,bar2,3\n", []interface{}{sql.Named("a", "foo1"), sql.Named("d", "foo3")}},
		{"With key()", "SELECT key(), a FROM test", false, "1,foo1\n2,foo2\n3\n", []interface{}{sql.Named("a", "foo1"), sql.Named("d", "foo3")}},
		{"With pk in cond, gt", "SELECT * FROM test WHERE k > 0 AND e = 1", false, "foo2,bar1,1,2\n", nil},
		{"With pk in cond, =", "SELECT * FROM test WHERE k = 2.0 AND e = 1", false, "foo2,bar1,1,2\n", nil},
	}

	for _, test := range tests {
		testFn := func(withIndexes bool) func(t *testing.T) {
			return func(t *testing.T) {
				db, err := genji.New(memoryengine.NewEngine())
				require.NoError(t, err)
				defer db.Close()

				err = db.Exec("CREATE TABLE test (k INTEGER PRIMARY KEY)")
				require.NoError(t, err)
				if withIndexes {
					err = db.Exec(`
						CREATE INDEX idx_a ON test (a);
						CREATE INDEX idx_b ON test (b);
						CREATE INDEX idx_c ON test (c);
						CREATE INDEX idx_d ON test (d);
					`)
					require.NoError(t, err)
				}

				err = db.Exec("INSERT INTO test (k, a, b, c) VALUES (1, 'foo1', 'bar1', 'baz1')")
				require.NoError(t, err)
				err = db.Exec("INSERT INTO test (k, a, b, e) VALUES (2, 'foo2', 'bar1', 1)")
				require.NoError(t, err)
				err = db.Exec("INSERT INTO test (k, d, e) VALUES (3, 'foo3', 'bar2')")
				require.NoError(t, err)

				st, err := db.Query(test.query, test.params...)
				if test.fails {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				defer st.Close()

				var buf bytes.Buffer
				err = document.IteratorToCSV(&buf, st)
				require.NoError(t, err)
				require.Equal(t, test.expected, buf.String())
			}
		}
		t.Run("No Index/"+test.name, testFn(false))
		t.Run("With Index/"+test.name, testFn(true))
	}

	t.Run("with primary key only", func(t *testing.T) {
		db, err := genji.New(memoryengine.NewEngine())
		require.NoError(t, err)
		defer db.Close()

		err = db.Exec("CREATE TABLE test (foo UINT8 PRIMARY KEY)")
		require.NoError(t, err)

		err = db.Exec(`INSERT INTO test (foo, bar) VALUES (1, 'a')`)
		err = db.Exec(`INSERT INTO test (foo, bar) VALUES (2, 'b')`)
		err = db.Exec(`INSERT INTO test (foo, bar) VALUES (3, 'c')`)
		err = db.Exec(`INSERT INTO test (foo, bar) VALUES (4, 'd')`)
		require.NoError(t, err)

		st, err := db.Query("SELECT * FROM test WHERE foo < 400 AND foo >= 2")
		require.NoError(t, err)
		defer st.Close()

		var buf bytes.Buffer
		err = document.IteratorToCSV(&buf, st)
		require.NoError(t, err)
		require.Equal(t, "b,2\nc,3\nd,4\n", buf.String())
	})

	t.Run("with documents", func(t *testing.T) {
		db, err := genji.New(memoryengine.NewEngine())
		require.NoError(t, err)
		defer db.Close()

		err = db.Exec("CREATE TABLE test")
		require.NoError(t, err)

		err = db.Exec(`INSERT INTO test VALUES {a: {b: 1}}`)
		require.NoError(t, err)

		st, err := db.Query("SELECT *, a.b FROM test WHERE a = {b: 1}")
		require.NoError(t, err)
		defer st.Close()

		var buf bytes.Buffer
		err = document.IteratorToJSON(&buf, st)
		require.NoError(t, err)
		require.JSONEq(t, `{"a": {"b":1}, "a.b": 1}`, buf.String())
	})
}