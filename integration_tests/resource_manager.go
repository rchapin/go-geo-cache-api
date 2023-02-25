package inttest

import (
	"context"
	"sync"
)

type ResourceManager struct {
	// Context, cancelfunc, for the test code itself
	tCtx    context.Context
	tCancel context.CancelFunc
	tWg     *sync.WaitGroup
	// Context, cancelfunc, and wg to be passed into the test runner
	testRunnerCtx    context.Context
	testRunnerCancel context.CancelFunc
	testRunnerWg     *sync.WaitGroup
}

func NewResourceManager() *ResourceManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResourceManager{
		tCtx:    ctx,
		tCancel: cancel,
		tWg:     &sync.WaitGroup{},
	}
}

func (r *ResourceManager) refreshContextsWg() {
	// For each test we need to create a new set of context, cancelfunc and wait groups to pass into
	// the TestRunner.  We generate the "sub" context/cancelfunc from the top level context that we
	// use to control the test suite itself.
	ctx, cancel := context.WithCancel(r.tCtx)
	r.testRunnerCtx = ctx
	r.testRunnerCancel = cancel
	r.testRunnerWg = &sync.WaitGroup{}
}
