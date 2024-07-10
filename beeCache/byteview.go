package beeCache

type ByteView struct {
	b []byte
}

// Len returns the length of the byte
func (view ByteView) Len() int {
	return len(view.b)
}

// ByteSlice returns a copy of the data as a byte slice.
func (view ByteView) ByteSlice() []byte {
	return cloneBytes(view.b)
}

// String returns the string of the byte data
func (view ByteView) String() string {
	return string(view.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
