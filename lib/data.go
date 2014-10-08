package lib

/*
 * This file contains data structures for drive-du.
 */

type Size int64

func (s Size) Pretty() string {
	return Pretty(int64(s))
}

type SizeEntry struct {
	Key   string
	Value Size
}

type BySize []SizeEntry

func (h BySize) Len() int           { return len(h) }
func (h BySize) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h BySize) Less(i, j int) bool { return h[i].Value > h[j].Value }

type ByName []SizeEntry

func (h ByName) Len() int           { return len(h) }
func (h ByName) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h ByName) Less(i, j int) bool { return h[i].Key < h[j].Key }
