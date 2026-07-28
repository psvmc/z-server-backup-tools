package service

import (
	"fmt"
	"math"

	"z-server-backup-tools/backend/model"
)

func formatMaxGBFlag(cfg model.BackupConfig) string {
	gb := cfg.MaxPartGB
	if gb <= 0 {
		gb = 2
	}
	return fmt.Sprintf("%g", gb)
}

func maxPartGBChanged(a, b float64) bool {
	return math.Abs(a-b) > 1e-6
}
