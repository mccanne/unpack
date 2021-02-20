package unpack_test

import (
	"encoding/json"
	"testing"

	"github.com/brimsec/zq/pkg/unpack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Expr interface {
	Which() string
}

type BinaryExpr struct {
	Op  string `json:"op"`
	LHS Expr   `json:"lhs"`
	RHS Expr   `json:"rhs"`
}

type UnaryExpr struct {
	Op      string `json:"op"`
	Operand Expr   `json:"operand"`
}

type ListExpr struct {
	Op    string `json:"op"`
	Exprs []Expr `json:"exprs"`
}

type Terminal struct {
	Body string `json:"body"`
}

func (t *Terminal) Which() string {
	return t.Body
}

func (*BinaryExpr) Which() string {
	return "BinaryExpr"
}

func (*UnaryExpr) Which() string {
	return "UnaryExpr"
}

const test1 = `
{
	"op":"BinaryExpr",
	"lhs": { "op": "Terminal", "body": "foo" } ,
	"rhs": { "op": "Terminal", "body": "bar" }
}`

var expected1 = &BinaryExpr{
	Op:  "BinaryExpr",
	LHS: &Terminal{"foo"},
	RHS: &Terminal{"bar"},
}

const test2 = `
{
	"op":"BinaryExpr",
	"lhs": {
		"op": "UnaryExpr",
		"operand": { "op": "Terminal",  "body": "foo" }
	},
	"rhs": {
		"op": "BinaryExpr",
		"lhs": { "op": "Terminal", "body": "bar" },
		"rhs": { "op": "Terminal", "body": "baz" }
	}
}`

var expected2 = &BinaryExpr{
	Op: "BinaryExpr",
	LHS: &UnaryExpr{
		Op:      "UnaryExpr",
		Operand: &Terminal{"foo"},
	},
	RHS: &BinaryExpr{
		Op:  "BinaryExpr",
		LHS: &Terminal{"bar"},
		RHS: &Terminal{"baz"},
	},
}

const test3 = `
{
	"op": "ListExpr",
	"exprs": [ {
		"op": "UnaryExpr",
		"operand": { "op": "Terminal",  "body": "foo" }
	},
	{
		"op": "BinaryExpr",
		"lhs": { "op": "Terminal", "body": "bar" },
		"rhs": { "op": "Terminal", "body": "baz" }
	} ]
}`

var expected3 = &ListExpr{
	Op: "ListExpr",
	Exprs: []Expr{
		&UnaryExpr{
			Op:      "UnaryExpr",
			Operand: &Terminal{"foo"},
		},
		&BinaryExpr{
			Op:  "BinaryExpr",
			LHS: &Terminal{"bar"},
			RHS: &Terminal{"baz"},
		},
	},
}

const test4 = `
{
	"op":"UnaryExpr",
	"operand": { "expr": { "op": "Terminal", "body": "nested" } }
}`

var expected4 = &UnaryExpr{
	Op:      "UnaryExpr",
	Operand: &Nested{&Terminal{"nested"}},
}

// Nesed is not an Op but has an interface value inside of it that can be
// recursively decoded.
type Nested struct {
	Expr Expr `json:"expr"`
}

func (n *Nested) Which() string {
	return "Nested"
}

func TestUnpack(t *testing.T) {
	var object interface{}
	err := json.Unmarshal([]byte(test1), &object)
	require.NoError(t, err)
	reflector := unpack.New().Init(
		BinaryExpr{},
		UnaryExpr{},
		Terminal{},
		ListExpr{},
	)
	result1, err := reflector.Unpack("op", test1)
	require.NoError(t, err)
	assert.Equal(t, result1, expected1)

	result2, err := reflector.Unpack("op", test2)
	require.NoError(t, err)
	assert.Equal(t, result2, expected2)

	// NOT YET
	//result3, err := reflector.Unpack("op", test3)
	//require.NoError(t, err)
	//assert.Equal(t, result3, expected3)

	result4, err := reflector.Unpack("op", test4)
	require.NoError(t, err)
	assert.Equal(t, result4, expected4)
}
