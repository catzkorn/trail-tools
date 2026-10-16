//go:build race

package images

// raceEnabled is whether the race detector is on. It makes sync.Pool drop
// some of the items put in it, so the x/image/draw scalers, which keep their
// buffer in one, reallocate it.
const raceEnabled = true
