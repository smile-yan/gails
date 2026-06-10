package commands

import (
	"github.com/gailsapp/gails/v3/internal/doctor"
)

type DoctorOptions struct{}

func Doctor(_ *DoctorOptions) error {
	return doctor.Run()
}
