// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godebug

import (
	"sync/atomic"
)

var goTypesAlias = &Setting{
	name: "gotypesalias",
}

func init() {
	goTypesAlias.Set("1")
}

// A Setting is a single setting in the $GODEBUG environment variable.
type Setting struct {
	name  string
	value atomic.Pointer[string]
}

func New(name string) *Setting {
	if len(name) != 0 && name[0] == '#' {
		name = name[1:]
	}
	switch name {
	case "gotypesalias":
		return goTypesAlias
	default:
		panic("unrechable")
	}
}

func (s *Setting) Name() string {
	if s.name != "" && s.name[0] == '#' {
		return s.name[1:]
	}
	return s.name
}

func (s *Setting) Undocumented() bool {
	return s.name != "" && s.name[0] == '#'
}

func (s *Setting) String() string {
	return s.Name() + "=" + s.Value()
}

func (s *Setting) IncNonDefault() {}

func (s *Setting) Value() string {
	return *s.value.Load()
}

func (s *Setting) Set(val string) {
	s.value.Store(&val)
}
