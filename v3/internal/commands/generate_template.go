package commands

import (
	"github.com/gailsapp/gails/v3/internal/templates"
)

func GenerateTemplate(options *templates.BaseTemplate) error {
	return templates.GenerateTemplate(options)
}
