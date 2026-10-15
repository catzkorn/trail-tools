//go:build race

package images

// raceEnabled is whether the race detector is on, which makes sync.Pool drop
// some of the items put in it, so the scalers reallocate their buffers.
const raceEnabled = true
