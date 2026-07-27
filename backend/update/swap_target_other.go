//go:build !darwin

package update

func swapTarget(exe string) string {
	return exe
}
