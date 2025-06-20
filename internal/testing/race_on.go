//go:build race
// +build race

package testing

func IsRaceOn() bool {
	return true
}
