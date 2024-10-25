// This file is derived from the sei-cosmos project.
// The original code is licensed under the Apache License, Version 2.0 (the "License");
//
// Modified for the Kaia development.

package multiversion

import (
	"sync"

	"github.com/google/btree"
	"github.com/kaiachain/kaia/common"
)

const (
	multiVersionBTreeDegree = 2
)

type MultiVersionValue interface {
	GetLatest() (value MultiVersionValueItem, found bool)
	GetLatestNonEstimate() (value MultiVersionValueItem, found bool)
	GetLatestBeforeIndex(index int) (value MultiVersionValueItem, found bool)
	Set(index int, incarnation int, value common.Hash)
	SetEstimate(index int, incarnation int)
	Delete(index int, incarnation int)
	Remove(index int)
}

type MultiVersionValueItem interface {
	IsDeleted() bool
	IsEstimate() bool
	Value() common.Hash
	Incarnation() int
	Index() int
}

type multiVersionItem struct {
	valueTree *btree.BTree // contains versions values written to this key
	mtx       sync.RWMutex // manages read + write accesses
}

var _ MultiVersionValue = (*multiVersionItem)(nil)

func NewMultiVersionItem() *multiVersionItem {
	return &multiVersionItem{
		valueTree: btree.New(multiVersionBTreeDegree),
	}
}

// GetLatest returns the latest written value to the btree, and returns a boolean indicating whether it was found.
func (item *multiVersionItem) GetLatest() (MultiVersionValueItem, bool) {
	item.mtx.RLock()
	bTreeItem := item.valueTree.Max()
	item.mtx.RUnlock()
	if bTreeItem == nil {
		return nil, false
	}
	valueItem := bTreeItem.(*valueItem)
	return valueItem, true
}

// GetLatestNonEstimate returns the latest written value that isn't an ESTIMATE and returns a boolean indicating whether it was found.
// This can be used when we want to write finalized values, since ESTIMATEs can be considered to be irrelevant at that point
func (item *multiVersionItem) GetLatestNonEstimate() (MultiVersionValueItem, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	var vItem *valueItem
	var found bool
	item.valueTree.Descend(func(bTreeItem btree.Item) bool {
		// only return if non-estimate
		item := bTreeItem.(*valueItem)
		if item.IsEstimate() {
			// if estimate, continue
			return true
		}
		// else we want to return
		vItem = item
		found = true
		return false
	})
	return vItem, found
}

// GetLatest returns the latest written value to the btree prior to the index passed in, and returns a boolean indicating whether it was found.
//
// A `nil` value along with `found=true` indicates a deletion that has occurred and the underlying parent store doesn't need to be hit.
func (item *multiVersionItem) GetLatestBeforeIndex(index int) (MultiVersionValueItem, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	// we want to find the value at the index that is LESS than the current index
	pivot := &valueItem{index: index - 1}

	var vItem *valueItem
	var found bool
	// start from pivot which contains our current index, and return on first item we hit.
	// This will ensure we get the latest indexed value relative to our current index
	item.valueTree.DescendLessOrEqual(pivot, func(bTreeItem btree.Item) bool {
		vItem = bTreeItem.(*valueItem)
		found = true
		return false
	})
	return vItem, found
}

func (item *multiVersionItem) Set(index int, incarnation int, value common.Hash) {
	if value == (common.Hash{}) {
		return
	}

	valueItem := NewValueItem(index, incarnation, value)
	item.mtx.Lock()
	defer item.mtx.Unlock()
	item.valueTree.ReplaceOrInsert(valueItem)
}

func (item *multiVersionItem) Delete(index int, incarnation int) {
	deletedItem := NewDeletedItem(index, incarnation)

	item.mtx.Lock()
	defer item.mtx.Unlock()
	item.valueTree.ReplaceOrInsert(deletedItem)
}

func (item *multiVersionItem) Remove(index int) {
	item.mtx.Lock()
	defer item.mtx.Unlock()

	item.valueTree.Delete(&valueItem{index: index})
}

func (item *multiVersionItem) SetEstimate(index int, incarnation int) {
	estimateItem := NewEstimateItem(index, incarnation)
	
	item.mtx.Lock()
	defer item.mtx.Unlock()
	item.valueTree.ReplaceOrInsert(estimateItem)
}

type valueItem struct {
	index       int
	incarnation int
	value       common.Hash
	estimate    bool
}

var _ MultiVersionValueItem = (*valueItem)(nil)

// Index implements MultiVersionValueItem.
func (v *valueItem) Index() int {
	return v.index
}

// Incarnation implements MultiVersionValueItem.
func (v *valueItem) Incarnation() int {
	return v.incarnation
}

// IsDeleted implements MultiVersionValueItem.
func (v *valueItem) IsDeleted() bool {
	return v.value == (common.Hash{}) && !v.estimate
}

// IsEstimate implements MultiVersionValueItem.
func (v *valueItem) IsEstimate() bool {
	return v.estimate
}

// Value implements MultiVersionValueItem.
func (v *valueItem) Value() common.Hash {
	if v == nil {
		return common.Hash{}
	}
	return v.value
}

// implement Less for btree.Item for valueItem
func (i *valueItem) Less(other btree.Item) bool {
	return i.index < other.(*valueItem).index
}

func NewValueItem(index int, incarnation int, value common.Hash) *valueItem {
	return &valueItem{
		index:       index,
		incarnation: incarnation,
		value:       value,
		estimate:    false,
	}
}

func NewEstimateItem(index int, incarnation int) *valueItem {
	return &valueItem{
		index:       index,
		incarnation: incarnation,
		value:       (common.Hash{}),
		estimate:    true,
	}
}

func NewDeletedItem(index int, incarnation int) *valueItem {
	return &valueItem{
		index:       index,
		incarnation: incarnation,
		value:       (common.Hash{}),
		estimate:    false,
	}
}
