package workspaces

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) GenerateSlug(ctx context.Context, name string) string {

	// Convert the name to lowercase
	slug := strings.ToLower(name)
	if len(slug) > 50 {
		slug = slug[:50]
	}
	if len(slug) == 0 {
		slug = "workspace" + uuid.New().String()[:4]
	}

	validSlug := slug
	isValid := false

	for !isValid {
		if ok, err := s.SnapshotRepo.CheckWorkspaceSlugExists(ctx, validSlug); err != nil {
			s.LogChan <- "Error checking slug existence: " + err.Error()
		} else if !ok {
			isValid = true
		} else {
			// Append a random suffix to make the slug unique
			validSlug = slug + "-" + uuid.New().String()[:4]
		}
	}
	return validSlug
}
