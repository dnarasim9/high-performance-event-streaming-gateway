package auth

import (
	"testing"
)

func TestRBACAuthorizer_HasPermission_AdminRole(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "admin-user",
		Roles:  []string{string(RoleAdmin)},
	}

	if !authorizer.HasPermission(claims, PermissionIngestEvent) {
		t.Error("HasPermission() admin should have PermissionIngestEvent")
	}

	if !authorizer.HasPermission(claims, PermissionManageSchema) {
		t.Error("HasPermission() admin should have PermissionManageSchema")
	}

	if !authorizer.HasPermission(claims, PermissionManageAuth) {
		t.Error("HasPermission() admin should have PermissionManageAuth")
	}
}

func TestRBACAuthorizer_HasPermission_PublisherRole(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "publisher-user",
		Roles:  []string{string(RolePublisher)},
	}

	if !authorizer.HasPermission(claims, PermissionIngestEvent) {
		t.Error("HasPermission() publisher should have PermissionIngestEvent")
	}

	if authorizer.HasPermission(claims, PermissionSubscribeEvent) {
		t.Error("HasPermission() publisher should not have PermissionSubscribeEvent")
	}

	if authorizer.HasPermission(claims, PermissionManageSchema) {
		t.Error("HasPermission() publisher should not have PermissionManageSchema")
	}
}

func TestRBACAuthorizer_HasPermission_SubscriberRole(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "subscriber-user",
		Roles:  []string{string(RoleSubscriber)},
	}

	if !authorizer.HasPermission(claims, PermissionSubscribeEvent) {
		t.Error("HasPermission() subscriber should have PermissionSubscribeEvent")
	}

	if !authorizer.HasPermission(claims, PermissionReplayEvent) {
		t.Error("HasPermission() subscriber should have PermissionReplayEvent")
	}

	if authorizer.HasPermission(claims, PermissionIngestEvent) {
		t.Error("HasPermission() subscriber should not have PermissionIngestEvent")
	}
}

func TestRBACAuthorizer_HasPermission_ViewerRole(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "viewer-user",
		Roles:  []string{string(RoleViewer)},
	}

	if !authorizer.HasPermission(claims, PermissionViewMetrics) {
		t.Error("HasPermission() viewer should have PermissionViewMetrics")
	}

	if authorizer.HasPermission(claims, PermissionIngestEvent) {
		t.Error("HasPermission() viewer should not have PermissionIngestEvent")
	}
}

func TestRBACAuthorizer_HasPermission_NilClaims(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	if authorizer.HasPermission(nil, PermissionIngestEvent) {
		t.Error("HasPermission() with nil claims should return false")
	}
}

func TestRBACAuthorizer_HasPermission_MultipleRoles(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "multi-role-user",
		Roles:  []string{string(RolePublisher), string(RoleSubscriber)},
	}

	// Should have both publisher and subscriber permissions
	if !authorizer.HasPermission(claims, PermissionIngestEvent) {
		t.Error("HasPermission() multi-role user should have PermissionIngestEvent (from publisher)")
	}

	if !authorizer.HasPermission(claims, PermissionSubscribeEvent) {
		t.Error("HasPermission() multi-role user should have PermissionSubscribeEvent (from subscriber)")
	}

	if authorizer.HasPermission(claims, PermissionManageSchema) {
		t.Error("HasPermission() multi-role user should not have PermissionManageSchema")
	}
}

func TestRBACAuthorizer_HasPermission_UnknownRole(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "user",
		Roles:  []string{"unknown-role"},
	}

	if authorizer.HasPermission(claims, PermissionIngestEvent) {
		t.Error("HasPermission() with unknown role should return false")
	}
}

func TestRBACAuthorizer_HasAnyPermission(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "viewer-user",
		Roles:  []string{string(RoleViewer)},
	}

	// Viewer has ViewMetrics but not IngestEvent
	if !authorizer.HasAnyPermission(claims, PermissionIngestEvent, PermissionViewMetrics) {
		t.Error("HasAnyPermission() should return true when at least one permission exists")
	}

	// Viewer has neither of these
	if authorizer.HasAnyPermission(claims, PermissionIngestEvent, PermissionSubscribeEvent) {
		t.Error("HasAnyPermission() should return false when no permission exists")
	}
}

func TestRBACAuthorizer_HasAllPermissions(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "admin-user",
		Roles:  []string{string(RoleAdmin)},
	}

	// Admin should have all permissions
	if !authorizer.HasAllPermissions(claims, PermissionIngestEvent, PermissionSubscribeEvent, PermissionManageSchema) {
		t.Error("HasAllPermissions() admin should have all permissions")
	}

	// Viewer doesn't have all
	claims2 := &Claims{
		UserID: "viewer-user",
		Roles:  []string{string(RoleViewer)},
	}

	if authorizer.HasAllPermissions(claims2, PermissionViewMetrics, PermissionIngestEvent) {
		t.Error("HasAllPermissions() viewer should not have all permissions")
	}
}

func TestRBACAuthorizer_GetRolePermissions(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	permissions := authorizer.GetRolePermissions(RolePublisher)

	if len(permissions) != 2 {
		t.Errorf("GetRolePermissions() publisher returned %d permissions, want 2", len(permissions))
	}

	// Verify it's a copy and can't affect original
	permissions[0] = "fake-permission"
	afterModify := authorizer.GetRolePermissions(RolePublisher)
	if afterModify[0] == "fake-permission" {
		t.Error("GetRolePermissions() should return a copy to prevent external modification")
	}
}

func TestRBACAuthorizer_GetUserPermissions(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	claims := &Claims{
		UserID: "multi-role-user",
		Roles:  []string{string(RolePublisher), string(RoleViewer)},
	}

	permissions := authorizer.GetUserPermissions(claims)

	// Publisher has IngestEvent + ViewMetrics (2)
	// Viewer has ViewMetrics (1) - but it's a duplicate
	// So unique permissions should be 2
	if len(permissions) < 2 {
		t.Errorf("GetUserPermissions() returned %d permissions, want >= 2", len(permissions))
	}

	// Verify no duplicates
	permMap := make(map[Permission]bool)
	for _, p := range permissions {
		if permMap[p] {
			t.Errorf("GetUserPermissions() returned duplicate permission: %v", p)
		}
		permMap[p] = true
	}
}

func TestRBACAuthorizer_GetUserPermissions_NilClaims(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	permissions := authorizer.GetUserPermissions(nil)

	if len(permissions) != 0 {
		t.Errorf("GetUserPermissions() with nil claims returned %d permissions, want 0", len(permissions))
	}
}

func TestRBACAuthorizer_AddRolePermissions(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	// Viewer only has ViewMetrics initially
	if authorizer.HasPermission(&Claims{Roles: []string{string(RoleViewer)}}, PermissionIngestEvent) {
		t.Error("Viewer should not have IngestEvent permission initially")
	}

	// Add IngestEvent to Viewer
	authorizer.AddRolePermissions(RoleViewer, PermissionIngestEvent)

	if !authorizer.HasPermission(&Claims{Roles: []string{string(RoleViewer)}}, PermissionIngestEvent) {
		t.Error("Viewer should have IngestEvent permission after AddRolePermissions")
	}
}

func TestRBACAuthorizer_RemoveRolePermissions(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	// Admin has IngestEvent initially
	if !authorizer.HasPermission(&Claims{Roles: []string{string(RoleAdmin)}}, PermissionIngestEvent) {
		t.Error("Admin should have IngestEvent permission initially")
	}

	// Remove IngestEvent from Admin
	authorizer.RemoveRolePermissions(RoleAdmin, PermissionIngestEvent)

	if authorizer.HasPermission(&Claims{Roles: []string{string(RoleAdmin)}}, PermissionIngestEvent) {
		t.Error("Admin should not have IngestEvent permission after RemoveRolePermissions")
	}
}

func TestRBACAuthorizer_PermissionDuplicatePrevention(t *testing.T) {
	authorizer := NewRBACAuthorizer()

	// Add same permission twice
	authorizer.AddRolePermissions(RoleViewer, PermissionIngestEvent, PermissionIngestEvent)

	permissions := authorizer.GetRolePermissions(RoleViewer)

	// Count IngestEvent occurrences
	count := 0
	for _, p := range permissions {
		if p == PermissionIngestEvent {
			count++
		}
	}

	if count != 1 {
		t.Errorf("AddRolePermissions() created %d duplicates of IngestEvent, want 0", count-1)
	}
}
