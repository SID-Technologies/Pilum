// Copyright © 2022-2023 Obol Labs Inc. Licensed under the terms of a Business Source License 1.1

package errors_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/sid-technologies/pilum/lib/errors"

	"github.com/stretchr/testify/require"
)

func TestComparable(t *testing.T) {
	t.Parallel()
	require.False(t, reflect.TypeOf(errors.New("x")).Comparable())
}

func TestIs(t *testing.T) {
	t.Parallel()
	errX := errors.New("x")

	err1 := errors.New("1")
	err11 := errors.Wrap(err1, "w1")
	err111 := errors.Wrap(err11, "w2")

	require.Equal(t, "x", errX.Error())
	require.Equal(t, "1", err1.Error())
	require.Equal(t, "w1: 1", err11.Error())
	require.Equal(t, "w2: w1: 1", err111.Error())

	require.True(t, errors.Is(err1, err1))
	require.True(t, errors.Is(err11, err1))
	require.True(t, errors.Is(err111, err1))
	require.False(t, errors.Is(err1, err11))
	require.True(t, errors.Is(err11, err11))
	require.True(t, errors.Is(err111, err11))
	require.False(t, errors.Is(err1, err111))
	require.False(t, errors.Is(err11, err111))
	require.True(t, errors.Is(err111, err11))

	require.False(t, errors.Is(err111, errX))

	errIO1 := errors.Wrap(io.EOF, "w1")
	errIO11 := errors.Wrap(errIO1, "w2")

	require.Equal(t, "w1: EOF", errIO1.Error())
	require.Equal(t, "w2: w1: EOF", errIO11.Error())

	require.True(t, errors.Is(io.EOF, io.EOF))
	require.True(t, errors.Is(errIO1, io.EOF))
	require.True(t, errors.Is(errIO11, io.EOF))
	require.False(t, errors.Is(io.EOF, errIO1))
	require.True(t, errors.Is(errIO1, errIO1))
	require.True(t, errors.Is(errIO11, errIO1))
	require.False(t, errors.Is(io.EOF, errIO11))
	require.False(t, errors.Is(errIO1, errIO11))
	require.True(t, errors.Is(errIO11, errIO11))
	require.False(t, errors.Is(err111, errX))
}

func TestNewWithFormatting(t *testing.T) {
	t.Parallel()

	err := errors.New("error with value: %d", 42)
	require.Equal(t, "error with value: 42", err.Error())
}

func TestWrapPreservesAttributes(t *testing.T) {
	t.Parallel()

	err1 := errors.New("base error %s %s", "key", "value")
	err2 := errors.Wrap(err1, "wrapped")

	require.Contains(t, err2.Error(), "wrapped")
	require.Contains(t, err2.Error(), "base error key value")
}

func TestWrapNilPanics(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		_ = errors.Wrap(nil, "should panic")
	})
}

func TestUnwrap(t *testing.T) {
	t.Parallel()

	inner := errors.New("inner")
	outer := errors.Wrap(inner, "outer")

	unwrapped := errors.Unwrap(outer)
	require.NotNil(t, unwrapped)
	require.Contains(t, unwrapped.Error(), "inner")
}

func TestUnwrapNil(t *testing.T) {
	t.Parallel()

	result := errors.Unwrap(nil)
	require.Nil(t, result)
}

func TestAs(t *testing.T) {
	t.Parallel()

	err := errors.New("test error")
	wrapped := errors.Wrap(err, "wrapped")

	var target error
	require.True(t, errors.As(wrapped, &target))
	require.NotNil(t, target)
}

func TestAsWithNil(t *testing.T) {
	t.Parallel()

	var target error
	require.False(t, errors.As(nil, &target))
}

func TestErrorChaining(t *testing.T) {
	t.Parallel()

	err1 := errors.New("level 1")
	err2 := errors.Wrap(err1, "level 2")
	err3 := errors.Wrap(err2, "level 3")

	require.Equal(t, "level 3: level 2: level 1", err3.Error())
	require.True(t, errors.Is(err3, err1))
	require.True(t, errors.Is(err3, err2))
}

func TestNewWithMultipleAttributes(t *testing.T) {
	t.Parallel()

	err := errors.New("test %s=%s %s=%s", "key1", "val1", "key2", "val2")
	require.NotNil(t, err)
	require.Equal(t, "test key1=val1 key2=val2", err.Error())
}

func TestWrapWithAdditionalAttributes(t *testing.T) {
	t.Parallel()

	inner := io.EOF
	wrapped := errors.Wrap(inner, "context", "key", "value")

	require.True(t, errors.Is(wrapped, io.EOF))
	require.Contains(t, wrapped.Error(), "context")
}

func TestWrapStandardLibraryError(t *testing.T) {
	t.Parallel()

	wrapped := errors.Wrap(io.EOF, "wrapped EOF")

	require.True(t, errors.Is(wrapped, io.EOF))
	require.Equal(t, "wrapped EOF: EOF", wrapped.Error())
}
