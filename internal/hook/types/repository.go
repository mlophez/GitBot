package types

import (
	"errors"
	"net/url"
)

var (
	ErrEmptyField     = errors.New("field must not be empty")
	ErrFieldTooLong   = errors.New("field must not exceed 1024 characters")
	ErrInvalidRepoURL = errors.New("link must be a valid URL")
)

type Repository struct {
	name string
	workspace string
	project string
	link string
}

// Name returns the repository's name.
func (r *Repository) Name() string { return r.name }
// Workspace returns the repository's workspace.
func (r *Repository) Workspace() string { return r.workspace }
// Project returns the repository's project.
func (r *Repository) Project() string { return r.project }
// Link returns the repository's link.
func (r *Repository) Link() string { return r.link }

// NewRepository creates a new immutable Repository instance.
func NewRepository(name, workspace, project, link string) (*Repository, error) {
	if err := validateField(name); err != nil {
		return nil, wrapFieldError("name", err)
	}
	if err := validateField(workspace); err != nil {
		return nil, wrapFieldError("workspace", err)
	}
	if err := validateField(project); err != nil {
		return nil, wrapFieldError("project", err)
	}
	if err := validateField(link); err != nil {
		return nil, wrapFieldError("link", err)
	}
	if _, err := url.ParseRequestURI(link); err != nil {
		return nil, wrapFieldError("link", ErrInvalidRepoURL)
	}

	return &Repository{
		name:      name,
		workspace: workspace,
		project:   project,
		link:      link,
	}, nil
}

func validateField(value string) error {
	if value == "" {
		return ErrEmptyField
	}
	if len(value) > 1024 {
		return ErrFieldTooLong
	}
	return nil
}

func wrapFieldError(field string, err error) error {
	return errors.New("invalid field '" + field + "': " + err.Error())
}
