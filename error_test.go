// Copyright 2021-2024 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connect

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/internal/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestErrorNilUnderlying(t *testing.T) {
	t.Parallel()
	err := NewError(CodeUnknown, nil)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), CodeUnknown.String())
	assert.Equal(t, err.Code(), CodeUnknown)
	assert.Zero(t, err.Details())
	detail, detailErr := NewErrorDetail(&emptypb.Empty{})
	assert.Nil(t, detailErr)
	err.AddDetail(detail)
	assert.Equal(t, len(err.Details()), 1)
	assert.Equal(t, err.Details()[0].Type(), "google.protobuf.Empty")
	err.Meta().Set("Foo", "bar")
	assert.Equal(t, err.Meta().Get("Foo"), "bar")
	assert.Equal(t, CodeOf(err), CodeUnknown)
}

func TestErrorFormatting(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		NewError(CodeUnavailable, errors.New("")).Error(),
		CodeUnavailable.String(),
	)
	got := NewError(CodeUnavailable, errors.New("Foo")).Error()
	assert.True(t, strings.Contains(got, CodeUnavailable.String()))
	assert.True(t, strings.Contains(got, "Foo"))
}

func TestErrorCode(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf(
		"another: %w",
		NewError(CodeUnavailable, errors.New("foo")),
	)
	connectErr, ok := asError(err)
	assert.True(t, ok)
	assert.Equal(t, connectErr.Code(), CodeUnavailable)
}

func TestCodeOf(t *testing.T) {
	t.Parallel()
	assert.Equal(
		t,
		CodeOf(NewError(CodeUnavailable, errors.New("foo"))),
		CodeUnavailable,
	)
	assert.Equal(t, CodeOf(errors.New("foo")), CodeUnknown)
}

func TestErrorDetails(t *testing.T) {
	t.Parallel()
	second := durationpb.New(time.Second)
	detail, err := NewErrorDetail(second)
	assert.Nil(t, err)
	connectErr := NewError(CodeUnknown, errors.New("error with details"))
	assert.Zero(t, connectErr.Details())
	connectErr.AddDetail(detail)
	assert.Equal(t, len(connectErr.Details()), 1)
	unmarshaled, err := connectErr.Details()[0].Value()
	assert.Nil(t, err)
	assert.Equal(t, unmarshaled, proto.Message(second))
	secondBin, err := proto.Marshal(second)
	assert.Nil(t, err)
	assert.Equal(t, detail.Bytes(), secondBin)
}

func TestErrorIs(t *testing.T) {
	t.Parallel()
	// errors.New and fmt.Errorf return *errors.errorString. errors.Is
	// considers two *errors.errorStrings equal iff they have the same address.
	err := errors.New("oh no")
	assert.False(t, errors.Is(err, errors.New("oh no")))
	assert.True(t, errors.Is(err, err))
	// Our errors should have the same semantics. Note that we'd need to extend
	// the ErrorDetail interface to support value equality.
	connectErr := NewError(CodeUnavailable, err)
	assert.False(t, errors.Is(connectErr, NewError(CodeUnavailable, err)))
	assert.True(t, errors.Is(connectErr, connectErr))
}

func TestTypeNameFromURL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		url      string
		typeName string
	}{
		{
			name:     "no-prefix",
			url:      "foo.bar.Baz",
			typeName: "foo.bar.Baz",
		},
		{
			name:     "standard-prefix",
			url:      defaultAnyResolverPrefix + "foo.bar.Baz",
			typeName: "foo.bar.Baz",
		},
		{
			name:     "different-hostname",
			url:      "abc.com/foo.bar.Baz",
			typeName: "foo.bar.Baz",
		},
		{
			name:     "additional-path-elements",
			url:      defaultAnyResolverPrefix + "abc/def/foo.bar.Baz",
			typeName: "foo.bar.Baz",
		},
		{
			name:     "full-url",
			url:      "https://abc.com/abc/def/foo.bar.Baz",
			typeName: "foo.bar.Baz",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, typeNameFromURL(testCase.url), testCase.typeName)
		})
	}
}
