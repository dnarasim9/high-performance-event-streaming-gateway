package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Role represents a user role in the system.
type Role string

// Role constants define the available user roles in the system.
const (
	RoleAdmin      Role = "admin"
	RolePublisher  Role = "publisher"
	RoleSubscriber Role = "subscriber"
	RoleViewer     Role = "viewer"
)

// Permission represents an action permission in the system.
type Permission string

// Permission constants define the available permissions in the system.
const (
	PermissionIngestEvent    Permission = "ingest_event"
	PermissionSubscribeEvent Permission = "subscribe_event"
	PermissionReplayEvent    Permission = "replay_event"
	PermissionManageConsumer Permission = "manage_consumer"
	PermissionManageSchema   Permission = "manage_schema"
	PermissionViewMetrics    Permission = "view_metrics"
	PermissionManageAuth     Permission = "manage_auth"
)

// RolePermissions maps roles to their allowed permissions.
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionIngestEvent,
		PermissionSubscribeEvent,
		PermissionReplayEvent,
		PermissionManageConsumer,
		PermissionManageSchema,
		PermissionViewMetrics,
		PermissionManageAuth,
	},
	RolePublisher: {
		PermissionIngestEvent,
		PermissionViewMetrics,
	},
	RoleSubscriber: {
		PermissionSubscribeEvent,
		PermissionReplayEvent,
		PermissionViewMetrics,
	},
	RoleViewer: {
		PermissionViewMetrics,
	},
}

// RBACAuthorizer handles role-based access control.
type RBACAuthorizer struct {
	rolePermissions map[Role][]Permission
}

// NewRBACAuthorizer creates a new RBACAuthorizer.
func NewRBACAuthorizer() *RBACAuthorizer {
	return &RBACAuthorizer{
		rolePermissions: RolePermissions,
	}
}

// HasPermission checks if the claims have the required permission.
func (ra *RBACAuthorizer) HasPermission(claims *Claims, permission Permission) bool {
	if claims == nil {
		return false
	}

	// Check each role
	for _, roleStr := range claims.Roles {
		role := Role(roleStr)
		permissions, exists := ra.rolePermissions[role]
		if !exists {
			continue
		}

		// Check if permission is in role's permissions
		for _, p := range permissions {
			if p == permission {
				return true
			}
		}
	}

	return false
}

// HasAnyPermission checks if the claims have any of the required permissions.
func (ra *RBACAuthorizer) HasAnyPermission(claims *Claims, permissions ...Permission) bool {
	for _, permission := range permissions {
		if ra.HasPermission(claims, permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the claims have all required permissions.
func (ra *RBACAuthorizer) HasAllPermissions(claims *Claims, permissions ...Permission) bool {
	for _, permission := range permissions {
		if !ra.HasPermission(claims, permission) {
			return false
		}
	}
	return true
}

// UnaryInterceptor returns a gRPC unary interceptor for RBAC authorization.
// It requires the JWT UnaryInterceptor to be run first to add claims to context.
func (ra *RBACAuthorizer) UnaryInterceptor(requiredPermission Permission) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract claims from context
		claims, err := ExtractClaims(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "claims not found in context")
		}

		// Check permission
		if !ra.HasPermission(claims, requiredPermission) {
			return nil, status.Errorf(
				codes.PermissionDenied,
				"user does not have required permission: %s",
				requiredPermission,
			)
		}

		// Call handler
		return handler(ctx, req)
	}
}

// StreamInterceptor returns a gRPC stream interceptor for RBAC authorization.
// It requires the JWT StreamInterceptor to be run first to add claims to context.
func (ra *RBACAuthorizer) StreamInterceptor(requiredPermission Permission) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Extract claims from context
		claims, err := ExtractClaims(ss.Context())
		if err != nil {
			return status.Error(codes.Unauthenticated, "claims not found in context")
		}

		// Check permission
		if !ra.HasPermission(claims, requiredPermission) {
			return status.Errorf(
				codes.PermissionDenied,
				"user does not have required permission: %s",
				requiredPermission,
			)
		}

		// Call handler
		return handler(srv, ss)
	}
}

// AddRolePermissions adds or overrides permissions for a role.
func (ra *RBACAuthorizer) AddRolePermissions(role Role, permissions ...Permission) {
	existingPermissions := ra.rolePermissions[role]
	existingPermissions = append(existingPermissions, permissions...)

	// Remove duplicates
	uniquePermissions := make([]Permission, 0)
	seen := make(map[Permission]bool)
	for _, p := range existingPermissions {
		if !seen[p] {
			uniquePermissions = append(uniquePermissions, p)
			seen[p] = true
		}
	}

	ra.rolePermissions[role] = uniquePermissions
}

// RemoveRolePermissions removes specific permissions from a role.
func (ra *RBACAuthorizer) RemoveRolePermissions(role Role, permissions ...Permission) {
	existingPermissions := ra.rolePermissions[role]
	permissionsToRemove := make(map[Permission]bool)
	for _, p := range permissions {
		permissionsToRemove[p] = true
	}

	filtered := make([]Permission, 0)
	for _, p := range existingPermissions {
		if !permissionsToRemove[p] {
			filtered = append(filtered, p)
		}
	}

	ra.rolePermissions[role] = filtered
}

// GetRolePermissions returns the permissions for a role.
func (ra *RBACAuthorizer) GetRolePermissions(role Role) []Permission {
	permissions, exists := ra.rolePermissions[role]
	if !exists {
		return []Permission{}
	}

	// Return a copy to prevent external modification
	result := make([]Permission, len(permissions))
	copy(result, permissions)
	return result
}

// GetUserPermissions returns all permissions for a user based on their roles.
func (ra *RBACAuthorizer) GetUserPermissions(claims *Claims) []Permission {
	if claims == nil {
		return []Permission{}
	}

	permissionSet := make(map[Permission]bool)
	for _, roleStr := range claims.Roles {
		role := Role(roleStr)
		permissions := ra.GetRolePermissions(role)
		for _, p := range permissions {
			permissionSet[p] = true
		}
	}

	result := make([]Permission, 0, len(permissionSet))
	for p := range permissionSet {
		result = append(result, p)
	}
	return result
}
