package auth

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/inelson/finguard/internal/models"
	"github.com/inelson/finguard/internal/store"
)

type RBAC struct {
	store    store.Store
	disabled bool
}

func NewRBAC(st store.Store, disabled bool) *RBAC {
	return &RBAC{store: st, disabled: disabled}
}

// RequireProjectRole returns middleware that checks the user has at least the given role on the project.
// The project ID is extracted from the URL param "projectID".
// Role hierarchy: platform-admin > admin > editor > viewer
func (rb *RBAC) RequireProjectRole(minRole models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rb.disabled {
				next.ServeHTTP(w, r)
				return
			}

			session := UserFromContext(r.Context())
			if session == nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}

			projectID := chi.URLParam(r, "projectID")
			if projectID == "" {
				next.ServeHTTP(w, r)
				return
			}

			if rb.hasSufficientRole(r.Context(), session, projectID, minRole) {
				next.ServeHTTP(w, r)
				return
			}

			writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
		})
	}
}

// RequirePlatformAdmin returns middleware that only allows platform-admin users.
func (rb *RBAC) RequirePlatformAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rb.disabled {
				next.ServeHTTP(w, r)
				return
			}

			session := UserFromContext(r.Context())
			if session == nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}

			if rb.isPlatformAdmin(r.Context(), session) {
				next.ServeHTTP(w, r)
				return
			}

			writeJSON(w, http.StatusForbidden, map[string]string{"error": "platform admin required"})
		})
	}
}

func (rb *RBAC) hasSufficientRole(ctx context.Context, session *SessionData, projectID string, minRole models.Role) bool {
	if rb.isPlatformAdmin(ctx, session) {
		return true
	}

	pr, err := rb.store.GetUserProjectRole(ctx, projectID, session.UserID)
	if err != nil || pr == nil {
		for _, groupClaim := range session.Groups {
			group, err := rb.store.GetGroupByOIDCClaim(ctx, groupClaim)
			if err != nil || group == nil {
				continue
			}
			roles, err := rb.store.ListProjectRoles(ctx, projectID)
			if err != nil {
				continue
			}
			for _, role := range roles {
				if role.SubjectType == models.SubjectGroup && role.SubjectID == group.ID {
					if roleLevel(role.Role) >= roleLevel(minRole) {
						return true
					}
				}
			}
		}
		return false
	}

	return roleLevel(pr.Role) >= roleLevel(minRole)
}

func (rb *RBAC) isPlatformAdmin(ctx context.Context, session *SessionData) bool {
	roles, err := rb.store.ListProjectRoles(ctx, "_global")
	if err != nil {
		return false
	}
	for _, role := range roles {
		if role.Role == models.RolePlatformAdmin && role.SubjectType == models.SubjectUser && role.SubjectID == session.UserID {
			return true
		}
	}
	return false
}

func roleLevel(role models.Role) int {
	switch role {
	case models.RolePlatformAdmin:
		return 100
	case models.RoleAdmin:
		return 30
	case models.RoleEditor:
		return 20
	case models.RoleViewer:
		return 10
	default:
		return 0
	}
}
