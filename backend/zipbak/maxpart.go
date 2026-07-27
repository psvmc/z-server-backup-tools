package zipbak

// MaxPartBytesFromGB converts client max_part_gb to bytes (default 2 GiB).
func MaxPartBytesFromGB(maxGB float64) int64 {
	if maxGB <= 0 {
		return 2 << 30
	}
	return int64(maxGB * (1 << 30))
}
