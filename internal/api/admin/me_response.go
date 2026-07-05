package admin

import (
	"context"

	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/security"
)

func (h *Handler) buildMeResponse(ctx context.Context, user *db.User, role string) (meResponse, error) {
	profile, err := h.DB.GetUserSecurityProfile(ctx, user.ID)
	if err != nil {
		return meResponse{}, err
	}
	sec, err := security.Load(ctx, h.DB)
	if err != nil {
		return meResponse{}, err
	}
	count, err := h.DB.CountUserSessions(ctx, user.ID)
	if err != nil {
		return meResponse{}, err
	}
	return meResponse{
		ID:                 user.ID,
		Email:              user.Email,
		DisplayName:        user.DisplayName,
		Role:               role,
		IsSystemAdmin:      user.IsSystemAdmin,
		HasPassword:        profile.HasPassword,
		AuthProvider:       profile.AuthProvider,
		TwoFAEnabled:       profile.TwoFAEnabled,
		TwoFAAvailable:     sec.TwoFA.AllowUserEnable,
		ActiveSessionCount: count,
	}, nil
}