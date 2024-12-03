package data

import (
	"strings"

	"greenlight.usman.com/internal/validator"
)

// page, page_size and sort query parameters are things that
// you'll potentially would want to use on other endpoints
// as well
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	// check if the page and page_size parameters contain
	// sensible values
	v.Check(f.Page > 0, "page", "must be greater than 0")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.Page <= 100, "page_size", "must be a maximum of 100")

	// Check that the sort parameter matches a value in the safelist
	v.Check(validator.PermittedValue(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

// Helper functions to get the sortColumn and sortDirection
func (f *Filters) sortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}
	panic("unsafe sort parameter: " + f.Sort)
}

func (f *Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

// Helpers for pagination
func (f *Filters) limit() int {
	return f.PageSize
}

func (f *Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}
