package stdjava

// ShuffleList performs Collections.shuffle's descending Fisher-Yates swaps.
// Access through List methods preserves a fixed-size list's array backing.
func ShuffleList[T any](list *List[T], supplied ...*Random) {
	ReferenceRequireNonNull(list)
	var random *Random
	if len(supplied) == 0 {
		random = NewRandom()
	} else {
		random = supplied[0]
	}
	for size := list.Size(); size > 1; size-- {
		ReferenceRequireNonNull(random)
		index := random.NextInt(size)
		previous := list.Set(index, list.Get(size-1))
		list.Set(size-1, previous)
	}
}
