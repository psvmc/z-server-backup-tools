package zipbak

// ParseStatusJSON decodes zipbak-srv status command stdout.
func ParseStatusJSON(out string) (Status, error) {
	return parseStatusJSON(out)
}
