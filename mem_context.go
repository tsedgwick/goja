package goja

import (
	"errors"
	"hash/maphash"
)

type visitTracker struct {
	objsVisited    map[objectImpl]bool
	stashesVisited map[*stash]bool
	valsVisited    map[uint64]bool
	h              *maphash.Hash
}

func (vt visitTracker) IsObjVisited(obj objectImpl) bool {
	_, ok := vt.objsVisited[obj]
	return ok
}

func (vt visitTracker) VisitObj(obj objectImpl) {
	vt.objsVisited[obj] = true
}

func (vt visitTracker) IsValVisited(obj Value) bool {
	if obj == nil {
		return true
	}
	_, ok := vt.valsVisited[obj.hash(vt.h)]
	return ok
}

func (vt visitTracker) VisitVal(obj Value) {
	vt.valsVisited[obj.hash(vt.h)] = true
}

func (vt visitTracker) IsStashVisited(stash *stash) bool {
	_, ok := vt.stashesVisited[stash]
	return ok
}

// func (vt visitTracker) IsStackVisited(stash valueStack) bool {
// 	_, ok := vt.stashesVisited[stash]
// 	fmt.Println("visited :check:")
// 	return ok
// }

func (vt visitTracker) VisitStash(stash *stash) {
	vt.stashesVisited[stash] = true
}

type depthTracker struct {
	curDepth int
	maxDepth int
}

func (dt depthTracker) Depth() int {
	return dt.curDepth
}

func (dt *depthTracker) Descend() error {
	if dt.curDepth == dt.maxDepth {
		return ErrMaxDepth
	}
	dt.curDepth++
	return nil
}

func (dt *depthTracker) Ascend() {
	if dt.curDepth == 0 {
		panic("can't ascend with depth 0")
	}
	dt.curDepth--
}

type NativeMemUsageChecker interface {
	NativeMemUsage(goNativeValue interface{}) (uint64, bool)
}

func (self *stash) MemUsage(ctx *MemUsageContext) (uint64, error) {
	if ctx.IsStashVisited(self) {
		return 0, nil
	}
	ctx.VisitStash(self)
	total := uint64(0)
	if self.obj != nil {
		inc, err := self.obj.MemUsage(ctx)
		total += inc
		if err != nil {
			return total, err
		}
	}

	if self.outer != nil {
		inc, err := self.outer.MemUsage(ctx)
		total += inc
		if err != nil {
			return total, err
		}
	}
	if len(self.values) > 0 {
		inc, err := self.values.MemUsage(ctx)
		total += inc
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

type MemUsageContext struct {
	vm *Runtime
	visitTracker
	*depthTracker
	NativeMemUsageChecker
}

func NewMemUsageContext(vm *Runtime, maxDepth int, nativeChecker NativeMemUsageChecker) *MemUsageContext {
	return &MemUsageContext{
		vm:                    vm,
		visitTracker:          visitTracker{objsVisited: map[objectImpl]bool{}, valsVisited: map[uint64]bool{}, stashesVisited: map[*stash]bool{}, h: &maphash.Hash{}},
		depthTracker:          &depthTracker{curDepth: 0, maxDepth: maxDepth},
		NativeMemUsageChecker: nativeChecker,
	}
}

var (
	ErrMaxDepth = errors.New("reached max depth")
)