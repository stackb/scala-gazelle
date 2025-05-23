// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	flag "flag"

	config "github.com/bazelbuild/bazel-gazelle/config"

	mock "github.com/stretchr/testify/mock"

	resolver "github.com/stackb/scala-gazelle/pkg/resolver"

	rule "github.com/bazelbuild/bazel-gazelle/rule"

	zerolog "github.com/rs/zerolog"
)

// ConflictResolver is an autogenerated mock type for the ConflictResolver type
type ConflictResolver struct {
	mock.Mock
}

// CheckFlags provides a mock function with given fields: fs, c
func (_m *ConflictResolver) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	ret := _m.Called(fs, c)

	if len(ret) == 0 {
		panic("no return value specified for CheckFlags")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*flag.FlagSet, *config.Config) error); ok {
		r0 = rf(fs, c)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with no fields
func (_m *ConflictResolver) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// RegisterFlags provides a mock function with given fields: fs, cmd, c, logger
func (_m *ConflictResolver) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config, logger zerolog.Logger) {
	_m.Called(fs, cmd, c, logger)
}

// ResolveConflict provides a mock function with given fields: universe, r, imports, imp, symbol
func (_m *ConflictResolver) ResolveConflict(universe resolver.Universe, r *rule.Rule, imports resolver.ImportMap, imp *resolver.Import, symbol *resolver.Symbol) (*resolver.Symbol, bool) {
	ret := _m.Called(universe, r, imports, imp, symbol)

	if len(ret) == 0 {
		panic("no return value specified for ResolveConflict")
	}

	var r0 *resolver.Symbol
	var r1 bool
	if rf, ok := ret.Get(0).(func(resolver.Universe, *rule.Rule, resolver.ImportMap, *resolver.Import, *resolver.Symbol) (*resolver.Symbol, bool)); ok {
		return rf(universe, r, imports, imp, symbol)
	}
	if rf, ok := ret.Get(0).(func(resolver.Universe, *rule.Rule, resolver.ImportMap, *resolver.Import, *resolver.Symbol) *resolver.Symbol); ok {
		r0 = rf(universe, r, imports, imp, symbol)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*resolver.Symbol)
		}
	}

	if rf, ok := ret.Get(1).(func(resolver.Universe, *rule.Rule, resolver.ImportMap, *resolver.Import, *resolver.Symbol) bool); ok {
		r1 = rf(universe, r, imports, imp, symbol)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// NewConflictResolver creates a new instance of ConflictResolver. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewConflictResolver(t interface {
	mock.TestingT
	Cleanup(func())
}) *ConflictResolver {
	mock := &ConflictResolver{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
