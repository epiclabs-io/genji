package document_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/asdine/genji/document"
	"github.com/asdine/genji/document/encoding"
	"github.com/stretchr/testify/require"
)

var _ document.Document = new(document.FieldBuffer)

func TestFieldBuffer(t *testing.T) {
	var buf document.FieldBuffer
	buf.Add("a", document.NewInt64Value(10))
	buf.Add("b", document.NewStringValue("hello"))

	t.Run("Iterate", func(t *testing.T) {
		var i int
		err := buf.Iterate(func(f string, v document.Value) error {
			switch i {
			case 0:
				require.Equal(t, "a", f)
				require.Equal(t, document.NewInt64Value(10), v)
			case 1:
				require.Equal(t, "b", f)
				require.Equal(t, document.NewStringValue("hello"), v)
			}
			i++
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, i)
	})

	t.Run("Add", func(t *testing.T) {
		var buf document.FieldBuffer
		buf.Add("a", document.NewInt64Value(10))
		buf.Add("b", document.NewStringValue("hello"))

		c := document.NewBoolValue(true)
		buf.Add("c", c)
		require.Equal(t, 3, buf.Len())
	})

	t.Run("ScanDocument", func(t *testing.T) {
		var buf1, buf2 document.FieldBuffer

		buf1.Add("a", document.NewInt64Value(10))
		buf1.Add("b", document.NewStringValue("hello"))

		buf2.Add("a", document.NewInt64Value(20))
		buf2.Add("b", document.NewStringValue("bye"))
		buf2.Add("c", document.NewBoolValue(true))

		err := buf1.ScanDocument(buf2)
		require.NoError(t, err)

		var buf document.FieldBuffer
		buf.Add("a", document.NewInt64Value(10))
		buf.Add("b", document.NewStringValue("hello"))
		buf.Add("a", document.NewInt64Value(20))
		buf.Add("b", document.NewStringValue("bye"))
		buf.Add("c", document.NewBoolValue(true))
		require.Equal(t, buf, buf1)
	})

	t.Run("GetByField", func(t *testing.T) {
		v, err := buf.GetByField("a")
		require.NoError(t, err)
		require.Equal(t, document.NewInt64Value(10), v)

		v, err = buf.GetByField("not existing")
		require.Equal(t, document.ErrFieldNotFound, err)
		require.Zero(t, v)
	})

	t.Run("Set", func(t *testing.T) {
		var buf document.FieldBuffer
		buf.Add("a", document.NewInt64Value(10))
		buf.Add("b", document.NewStringValue("hello"))

		buf.Set("a", document.NewFloat64Value(11))
		v, err := buf.GetByField("a")
		require.NoError(t, err)
		require.Equal(t, document.NewFloat64Value(11), v)

		buf.Set("c", document.NewInt64Value(12))
		require.Equal(t, 3, buf.Len())
		v, err = buf.GetByField("c")
		require.NoError(t, err)
		require.Equal(t, document.NewInt64Value(12), v)
	})

	t.Run("Delete", func(t *testing.T) {
		var buf document.FieldBuffer
		buf.Add("a", document.NewInt64Value(10))
		buf.Add("b", document.NewStringValue("hello"))

		err := buf.Delete("a")
		require.NoError(t, err)
		require.Equal(t, 1, buf.Len())
		v, _ := buf.GetByField("b")
		require.Equal(t, document.NewStringValue("hello"), v)
		_, err = buf.GetByField("a")
		require.Error(t, err)

		err = buf.Delete("b")
		require.NoError(t, err)
		require.Equal(t, 0, buf.Len())

		err = buf.Delete("b")
		require.Error(t, err)
	})

	t.Run("Replace", func(t *testing.T) {
		var buf document.FieldBuffer
		buf.Add("a", document.NewInt64Value(10))
		buf.Add("b", document.NewStringValue("hello"))

		err := buf.Replace("a", document.NewBoolValue(true))
		require.NoError(t, err)
		v, err := buf.GetByField("a")
		require.NoError(t, err)
		require.Equal(t, document.NewBoolValue(true), v)
		err = buf.Replace("d", document.NewInt64Value(11))
		require.Error(t, err)
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		tests := []struct {
			name     string
			data     string
			expected *document.FieldBuffer
			fails    bool
		}{
			{"empty object", "{}", document.NewFieldBuffer(), false},
			{"empty object, missing closing bracket", "{", nil, true},
			{"classic object", `{"a": 1, "b": true, "c": "hello", "d": [1, 2, 3], "e": {"f": "g"}}`,
				document.NewFieldBuffer().
					Add("a", document.NewInt8Value(1)).
					Add("b", document.NewBoolValue(true)).
					Add("c", document.NewStringValue("hello")).
					Add("d", document.NewArrayValue(document.NewValueBuffer().
						Append(document.NewInt8Value(1)).
						Append(document.NewInt8Value(2)).
						Append(document.NewInt8Value(3)))).
					Add("e", document.NewDocumentValue(document.NewFieldBuffer().Add("f", document.NewStringValue("g")))),
				false},
			{"string values", `{"a": "hello ciao"}`, document.NewFieldBuffer().Add("a", document.NewStringValue("hello ciao")), false},
			{"+int8 values", `{"a": 1}`, document.NewFieldBuffer().Add("a", document.NewInt8Value(1)), false},
			{"-int8 values", `{"a": -1}`, document.NewFieldBuffer().Add("a", document.NewInt8Value(-1)), false},
			{"+int16 values", `{"a": 1000}`, document.NewFieldBuffer().Add("a", document.NewInt16Value(1000)), false},
			{"-int16 values", `{"a": 1000}`, document.NewFieldBuffer().Add("a", document.NewInt16Value(1000)), false},
			{"+int32 values", `{"a": 1000000}`, document.NewFieldBuffer().Add("a", document.NewInt32Value(1000000)), false},
			{"-int32 values", `{"a": 1000000}`, document.NewFieldBuffer().Add("a", document.NewInt32Value(1000000)), false},
			{"+int64 values", `{"a": 10000000000}`, document.NewFieldBuffer().Add("a", document.NewInt64Value(10000000000)), false},
			{"-int64 values", `{"a": -10000000000}`, document.NewFieldBuffer().Add("a", document.NewInt64Value(-10000000000)), false},
			{"uint64 values", `{"a": 10000000000000000000}`, document.NewFieldBuffer().Add("a", document.NewUint64Value(10000000000000000000)), false},
			{"+float64 values", `{"a": 10000000000.0}`, document.NewFieldBuffer().Add("a", document.NewFloat64Value(10000000000)), false},
			{"-float64 values", `{"a": -10000000000.0}`, document.NewFieldBuffer().Add("a", document.NewFloat64Value(-10000000000)), false},
			{"bool values", `{"a": true, "b": false}`, document.NewFieldBuffer().Add("a", document.NewBoolValue(true)).Add("b", document.NewBoolValue(false)), false},
			{"empty arrays", `{"a": []}`, document.NewFieldBuffer().Add("a", document.NewArrayValue(document.NewValueBuffer())), false},
			{"nested arrays", `{"a": [[1,  2]]}`, document.NewFieldBuffer().
				Add("a", document.NewArrayValue(
					document.NewValueBuffer().
						Append(document.NewArrayValue(
							document.NewValueBuffer().
								Append(document.NewInt8Value(1)).
								Append(document.NewInt8Value(2)))))), false},
			{"missing comma", `{"a": 1 "b": 2}`, nil, true},
			{"missing closing brackets", `{"a": 1, "b": 2`, nil, true},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var buf document.FieldBuffer

				err := json.Unmarshal([]byte(test.data), &buf)
				if test.fails {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, *test.expected, buf)
				}
			})
		}
	})
}

func TestNewFromMap(t *testing.T) {
	m := map[string]interface{}{
		"name":     "foo",
		"age":      10,
		"nilField": nil,
	}

	doc := document.NewFromMap(m)

	t.Run("Iterate", func(t *testing.T) {
		counter := make(map[string]int)

		err := doc.Iterate(func(f string, v document.Value) error {
			counter[f]++
			switch f {
			case "name":
				require.Equal(t, m[f], string(v.V.([]byte)))
			default:
				require.Equal(t, m[f], v.V)
			}
			return nil
		})
		require.NoError(t, err)
		require.Len(t, counter, 3)
		require.Equal(t, counter["name"], 1)
		require.Equal(t, counter["age"], 1)
		require.Equal(t, counter["nilField"], 1)
	})

	t.Run("GetByField", func(t *testing.T) {
		v, err := doc.GetByField("name")
		require.NoError(t, err)
		require.Equal(t, document.NewStringValue("foo"), v)

		v, err = doc.GetByField("age")
		require.NoError(t, err)
		require.Equal(t, document.NewIntValue(10), v)

		v, err = doc.GetByField("nilField")
		require.NoError(t, err)
		require.Equal(t, document.NewNullValue(), v)

		_, err = doc.GetByField("bar")
		require.Equal(t, document.ErrFieldNotFound, err)
	})
}

func TestNewFromStruct(t *testing.T) {
	type group struct {
		A int
	}

	type user struct {
		A []byte
		B string
		C bool
		D uint `genji:"la-reponse-d"`
		E uint8
		F uint16
		G uint32
		H uint64
		I int
		J int8
		K int16
		L int32
		M int64
		N float64
		// structs must be considered as documents
		O group

		// nil pointers must be considered as Null values
		// otherwise they must be dereferenced
		P *int
		Q *int

		// struct pointers should be considered as documents
		// if there are nil though, the value must be Null
		R *group
		S *group

		T  []int
		U  []int
		V  []*int
		W  []user
		X  []interface{}
		Y  [3]int
		Z  interface{}
		ZZ interface{}

		AA int `genji:"-"` // ignored

		// embedded fields are not supported currently, they should be ignored
		*group

		// unexported fields should be ignored
		t int
	}

	u := user{
		A:  []byte("foo"),
		B:  "bar",
		C:  true,
		D:  1,
		E:  2,
		F:  3,
		G:  4,
		H:  5,
		I:  6,
		J:  7,
		K:  8,
		L:  9,
		M:  10,
		N:  11.12,
		Z:  26,
		AA: 27,
	}

	q := 5
	u.Q = &q
	u.R = new(group)
	u.T = []int{1, 2, 3}
	u.V = []*int{&q}
	u.W = []user{u}
	u.X = []interface{}{1, "foo"}

	t.Run("Iterate", func(t *testing.T) {
		doc, err := document.NewFromStruct(u)
		require.NoError(t, err)

		var counter int

		err = doc.Iterate(func(f string, v document.Value) error {
			switch counter {
			case 0:
				require.Equal(t, u.A, v.V.([]byte))
			case 1:
				require.Equal(t, u.B, string(v.V.([]byte)))
			case 2:
				require.Equal(t, u.C, v.V.(bool))
			case 3:
				require.Equal(t, "la-reponse-d", f)
				require.Equal(t, u.D, v.V.(uint))
			case 4:
				require.Equal(t, u.E, v.V.(uint8))
			case 5:
				require.Equal(t, u.F, v.V.(uint16))
			case 6:
				require.Equal(t, u.G, v.V.(uint32))
			case 7:
				require.Equal(t, u.H, v.V.(uint64))
			case 8:
				require.Equal(t, u.I, v.V.(int))
			case 9:
				require.Equal(t, u.J, v.V.(int8))
			case 10:
				require.Equal(t, u.K, v.V.(int16))
			case 11:
				require.Equal(t, u.L, v.V.(int32))
			case 12:
				require.Equal(t, u.M, v.V.(int64))
			case 13:
				require.Equal(t, u.N, v.V.(float64))
			case 14:
				require.Equal(t, document.DocumentValue, v.Type)
			case 15:
				require.Equal(t, document.NullValue, v.Type)
			case 16:
				require.Equal(t, *u.Q, v.V.(int))
			case 17:
				require.Equal(t, document.DocumentValue, v.Type)
			case 18:
				require.Equal(t, document.NullValue, v.Type)
			case 19:
				require.Equal(t, document.ArrayValue, v.Type)
			case 20:
				require.Equal(t, document.NullValue, v.Type)
			case 21:
				require.Equal(t, document.ArrayValue, v.Type)
			case 22:
				require.Equal(t, document.ArrayValue, v.Type)
			case 23:
				require.Equal(t, document.ArrayValue, v.Type)
			case 24:
				require.Equal(t, document.ArrayValue, v.Type)
			case 25:
				require.Equal(t, u.Z, v.V.(int))
			case 26:
				require.Equal(t, document.NullValue, v.Type)
			default:
				require.FailNowf(t, "", "unknown field %q", f)
			}

			counter++

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 27, counter)
	})

	t.Run("GetByField", func(t *testing.T) {
		doc, err := document.NewFromStruct(u)
		require.NoError(t, err)

		v, err := doc.GetByField("a")
		require.NoError(t, err)
		require.Equal(t, u.A, v.V.([]byte))
		v, err = doc.GetByField("b")
		require.NoError(t, err)
		require.Equal(t, u.B, string(v.V.([]byte)))
		v, err = doc.GetByField("c")
		require.NoError(t, err)
		require.Equal(t, u.C, v.V.(bool))
		v, err = doc.GetByField("la-reponse-d")
		require.NoError(t, err)
		require.Equal(t, u.D, v.V.(uint))
		v, err = doc.GetByField("e")
		require.NoError(t, err)
		require.Equal(t, u.E, v.V.(uint8))
		v, err = doc.GetByField("f")
		require.NoError(t, err)
		require.Equal(t, u.F, v.V.(uint16))
		v, err = doc.GetByField("g")
		require.NoError(t, err)
		require.Equal(t, u.G, v.V.(uint32))
		v, err = doc.GetByField("h")
		require.NoError(t, err)
		require.Equal(t, u.H, v.V.(uint64))
		v, err = doc.GetByField("i")
		require.NoError(t, err)
		require.Equal(t, u.I, v.V.(int))
		v, err = doc.GetByField("j")
		require.NoError(t, err)
		require.Equal(t, u.J, v.V.(int8))
		v, err = doc.GetByField("k")
		require.NoError(t, err)
		require.Equal(t, u.K, v.V.(int16))
		v, err = doc.GetByField("l")
		require.NoError(t, err)
		require.Equal(t, u.L, v.V.(int32))
		v, err = doc.GetByField("m")
		require.NoError(t, err)
		require.Equal(t, u.M, v.V.(int64))
		v, err = doc.GetByField("n")
		require.NoError(t, err)
		require.Equal(t, u.N, v.V.(float64))

		v, err = doc.GetByField("o")
		require.NoError(t, err)
		d, err := v.ConvertToDocument()
		require.NoError(t, err)
		v, err = d.GetByField("a")
		require.NoError(t, err)
		require.Equal(t, 0, v.V.(int))

		v, err = doc.GetByField("t")
		require.NoError(t, err)
		a, err := v.ConvertToArray()
		require.NoError(t, err)
		var count int
		err = a.Iterate(func(i int, v document.Value) error {
			count++
			require.Equal(t, i+1, v.V.(int))
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		v, err = a.GetByIndex(10)
		require.Equal(t, err, document.ErrFieldNotFound)
		v, err = a.GetByIndex(1)
		require.NoError(t, err)
		require.Equal(t, 2, v.V.(int))
	})
}

type foo struct {
	A string
	B int
	C bool
	D float64
}

func (f *foo) Iterate(fn func(field string, value document.Value) error) error {
	var err error

	err = fn("a", document.NewStringValue(f.A))
	if err != nil {
		return err
	}

	err = fn("b", document.NewIntValue(f.B))
	if err != nil {
		return err
	}

	err = fn("c", document.NewBoolValue(f.C))
	if err != nil {
		return err
	}

	err = fn("d", document.NewFloat64Value(f.D))
	if err != nil {
		return err
	}

	return nil
}

func (f *foo) GetByField(field string) (document.Value, error) {
	switch field {
	case "a":
		return document.NewStringValue(f.A), nil
	case "b":
		return document.NewIntValue(f.B), nil
	case "c":
		return document.NewBoolValue(f.C), nil
	case "d":
		return document.NewFloat64Value(f.D), nil
	}

	return document.Value{}, errors.New("unknown field")
}

func TestValuePath(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		path   string
		result string
		fails  bool
	}{
		{"empty path", `{"a": 1}`, ``, ``, true},
		{"root", `{"a": {"b": [1, 2, 3]}}`, `a`, `{"b": [1, 2, 3]}`, false},
		{"nested doc", `{"a": {"b": [1, 2, 3]}}`, `a.b`, `[1, 2, 3]`, false},
		{"nested array", `{"a": {"b": [1, 2, 3]}}`, `a.b.1`, `2`, false},
		{"index out of range", `{"a": {"b": [1, 2, 3]}}`, `a.b.1000`, ``, true},
		{"number field", `{"a": {"0": [1, 2, 3]}}`, `a.0`, `[1, 2, 3]`, false},
		{"letter index", `{"a": {"b": [1, 2, 3]}}`, `a.b.c`, ``, true},
		{"unknown path", `{"a": {"b": [1, 2, 3]}}`, `a.e.f`, ``, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf document.FieldBuffer

			err := json.Unmarshal([]byte(test.data), &buf)
			require.NoError(t, err)
			p := document.NewValuePath(test.path)
			v, err := p.GetValue(&buf)
			if test.fails {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				res, err := json.Marshal(v)
				require.NoError(t, err)
				require.JSONEq(t, test.result, string(res))
			}
		})
	}
}

func BenchmarkDocumentIterate(b *testing.B) {
	f := foo{
		A: "a",
		B: 1000,
		C: true,
		D: 1e10,
	}

	b.Run("Implementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f.Iterate(func(string, document.Value) error {
				return nil
			})
		}
	})

	b.Run("Reflection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			refd, _ := document.NewFromStruct(&f)
			refd.Iterate(func(string, document.Value) error {
				return nil
			})
		}
	})

	b.Run("Encoding/Implementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoding.EncodeDocument(&f)
		}
	})

	b.Run("Encoding/Reflection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			refd, _ := document.NewFromStruct(&f)
			encoding.EncodeDocument(refd)
		}
	})

	b.Run("Encoding/JSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			json.Marshal(&f)
		}
	})
}
