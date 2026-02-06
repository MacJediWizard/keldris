// Package invites provides user invitation management services.
package invites

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DefaultExpiryDuration is the default expiry for invitations (7 days).
const DefaultExpiryDuration = 7 * 24 * time.Hour

// MaxBulkInvites is the maximum number of invitations allowed in a single bulk operation.
const MaxBulkInvites = 100

// Store defines the interface for invite persistence operations.
type Store interface {
	// Invitation operations
	CreateInvitation(ctx context.Context, inv *models.OrgInvitation) error
	GetInvitationByID(ctx context.Context, id uuid.UUID) (*models.OrgInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*models.OrgInvitation, error)
	GetPendingInvitationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.OrgInvitationWithDetails, error)
	GetPendingInvitationsByEmail(ctx context.Context, email string) ([]*models.OrgInvitationWithDetails, error)
	AcceptInvitation(ctx context.Context, id uuid.UUID) error
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
	UpdateInvitationResent(ctx context.Context, id uuid.UUID) error

	// Membership operations
	CreateMembership(ctx context.Context, m *models.OrgMembership) error
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)

	// Organization operations
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)

	// User operations
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
}

// InviteRequest represents a request to create an invitation.
type InviteRequest struct {
	Email     string         `json:"email" binding:"required,email"`
	Role      models.OrgRole `json:"role" binding:"required"`
	OrgID     uuid.UUID      `json:"-"`
	InvitedBy uuid.UUID      `json:"-"`
}

// BulkInviteRequest represents a request to create multiple invitations.
type BulkInviteRequest struct {
	OrgID     uuid.UUID
	InvitedBy uuid.UUID
	Invites   []InviteRequest
}

// BulkInviteResult represents the result of a bulk invitation operation.
type BulkInviteResult struct {
	Successful []InviteResult `json:"successful"`
	Failed     []InviteError  `json:"failed"`
	Total      int            `json:"total"`
}

// InviteResult represents a successful invitation.
type InviteResult struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	Token string `json:"token,omitempty"`
}

// InviteError represents a failed invitation.
type InviteError struct {
	Email string `json:"email"`
	Error string `json:"error"`
}

// Service handles invitation operations.
type Service struct {
	store        Store
	emailService *notifications.EmailService
	baseURL      string
	logger       zerolog.Logger
}

// NewService creates a new invite service.
func NewService(store Store, emailService *notifications.EmailService, baseURL string, logger zerolog.Logger) *Service {
	return &Service{
		store:        store,
		emailService: emailService,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		logger:       logger.With().Str("component", "invite_service").Logger(),
	}
}

// GenerateToken generates a secure random token for invitations.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateInviteLink generates the full invitation URL.
func (s *Service) GenerateInviteLink(token string) string {
	return fmt.Sprintf("%s/invite/%s", s.baseURL, token)
}

// CreateInvitation creates a new invitation and optionally sends an email.
func (s *Service) CreateInvitation(ctx context.Context, req InviteRequest, sendEmail bool) (*models.OrgInvitation, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Validate role
	if !models.IsValidOrgRole(string(req.Role)) {
		return nil, fmt.Errorf("invalid role: %s", req.Role)
	}

	// Check if user is already a member
	existingUser, err := s.store.GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		membership, err := s.store.GetMembershipByUserAndOrg(ctx, existingUser.ID, req.OrgID)
		if err == nil && membership != nil {
			return nil, fmt.Errorf("user is already a member of this organization")
		}
	}

	// Check for existing pending invitation
	pending, err := s.store.GetPendingInvitationsByEmail(ctx, email)
	if err == nil {
		for _, inv := range pending {
			if inv.OrgID == req.OrgID && time.Now().Before(inv.ExpiresAt) && inv.AcceptedAt == nil {
				return nil, fmt.Errorf("a pending invitation already exists for this email")
			}
		}
	}

	// Generate token
	token, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Create invitation
	inv := models.NewOrgInvitation(
		req.OrgID,
		email,
		req.Role,
		token,
		req.InvitedBy,
		time.Now().Add(DefaultExpiryDuration),
	)

	if err := s.store.CreateInvitation(ctx, inv); err != nil {
		return nil, fmt.Errorf("store invitation: %w", err)
	}

	s.logger.Info().
		Str("invitation_id", inv.ID.String()).
		Str("email", email).
		Str("role", string(req.Role)).
		Str("org_id", req.OrgID.String()).
		Str("invited_by", req.InvitedBy.String()).
		Msg("invitation created")

	// Send email if requested
	if sendEmail && s.emailService != nil {
		if err := s.sendInvitationEmail(ctx, inv); err != nil {
			s.logger.Warn().Err(err).
				Str("email", email).
				Msg("failed to send invitation email")
			// Don't fail the whole operation if email fails
		}
	}

	return inv, nil
}

// ResendInvitation resends an invitation email.
func (s *Service) ResendInvitation(ctx context.Context, invitationID uuid.UUID) error {
	inv, err := s.store.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("get invitation: %w", err)
	}

	if inv.IsAccepted() {
		return fmt.Errorf("invitation has already been accepted")
	}

	if inv.IsExpired() {
		return fmt.Errorf("invitation has expired")
	}

	// Update resent timestamp
	if err := s.store.UpdateInvitationResent(ctx, invitationID); err != nil {
		s.logger.Warn().Err(err).Msg("failed to update resent timestamp")
	}

	// Send email
	if s.emailService != nil {
		if err := s.sendInvitationEmail(ctx, inv); err != nil {
			return fmt.Errorf("send email: %w", err)
		}
	}

	s.logger.Info().
		Str("invitation_id", invitationID.String()).
		Str("email", inv.Email).
		Msg("invitation resent")

	return nil
}

// RevokeInvitation revokes a pending invitation.
func (s *Service) RevokeInvitation(ctx context.Context, invitationID uuid.UUID, revokedBy uuid.UUID) error {
	inv, err := s.store.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("get invitation: %w", err)
	}

	if inv.IsAccepted() {
		return fmt.Errorf("cannot revoke an accepted invitation")
	}

	if err := s.store.DeleteInvitation(ctx, invitationID); err != nil {
		return fmt.Errorf("delete invitation: %w", err)
	}

	s.logger.Info().
		Str("invitation_id", invitationID.String()).
		Str("email", inv.Email).
		Str("revoked_by", revokedBy.String()).
		Msg("invitation revoked")

	return nil
}

// AcceptInvitation accepts an invitation and creates the organization membership.
func (s *Service) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.Organization, error) {
	inv, err := s.store.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invitation not found")
	}

	if inv.IsExpired() {
		return nil, fmt.Errorf("invitation has expired")
	}

	if inv.IsAccepted() {
		return nil, fmt.Errorf("invitation has already been accepted")
	}

	// Get user to verify email matches
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !strings.EqualFold(user.Email, inv.Email) {
		return nil, fmt.Errorf("invitation is for a different email address")
	}

	// Check if already a member
	existing, _ := s.store.GetMembershipByUserAndOrg(ctx, userID, inv.OrgID)
	if existing != nil {
		return nil, fmt.Errorf("already a member of this organization")
	}

	// Create membership
	membership := models.NewOrgMembership(userID, inv.OrgID, inv.Role)
	if err := s.store.CreateMembership(ctx, membership); err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}

	// Mark invitation as accepted
	if err := s.store.AcceptInvitation(ctx, inv.ID); err != nil {
		s.logger.Warn().Err(err).Msg("failed to mark invitation as accepted")
	}

	// Get organization
	org, err := s.store.GetOrganizationByID(ctx, inv.OrgID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to get organization")
		return nil, nil
	}

	s.logger.Info().
		Str("invitation_id", inv.ID.String()).
		Str("user_id", userID.String()).
		Str("org_id", inv.OrgID.String()).
		Str("role", string(inv.Role)).
		Msg("invitation accepted")

	return org, nil
}

// GetInvitationByToken retrieves an invitation by its token (for public viewing).
func (s *Service) GetInvitationByToken(ctx context.Context, token string) (*models.OrgInvitationWithDetails, error) {
	inv, err := s.store.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invitation not found")
	}

	org, err := s.store.GetOrganizationByID(ctx, inv.OrgID)
	if err != nil {
		return nil, fmt.Errorf("organization not found")
	}

	inviter, err := s.store.GetUserByID(ctx, inv.InvitedBy)
	inviterName := "Unknown"
	if err == nil && inviter != nil {
		inviterName = inviter.Name
		if inviterName == "" {
			inviterName = inviter.Email
		}
	}

	return &models.OrgInvitationWithDetails{
		ID:          inv.ID,
		OrgID:       inv.OrgID,
		OrgName:     org.Name,
		Email:       inv.Email,
		Role:        inv.Role,
		InvitedBy:   inv.InvitedBy,
		InviterName: inviterName,
		ExpiresAt:   inv.ExpiresAt,
		AcceptedAt:  inv.AcceptedAt,
		CreatedAt:   inv.CreatedAt,
	}, nil
}

// GetPendingInvitations retrieves all pending invitations for an organization.
func (s *Service) GetPendingInvitations(ctx context.Context, orgID uuid.UUID) ([]*models.OrgInvitationWithDetails, error) {
	return s.store.GetPendingInvitationsByOrgID(ctx, orgID)
}

// BulkInvite creates multiple invitations from a list.
func (s *Service) BulkInvite(ctx context.Context, req BulkInviteRequest, sendEmails bool) (*BulkInviteResult, error) {
	if len(req.Invites) > MaxBulkInvites {
		return nil, fmt.Errorf("maximum %d invitations allowed per batch", MaxBulkInvites)
	}

	result := &BulkInviteResult{
		Successful: make([]InviteResult, 0),
		Failed:     make([]InviteError, 0),
		Total:      len(req.Invites),
	}

	for _, invite := range req.Invites {
		invite.OrgID = req.OrgID
		invite.InvitedBy = req.InvitedBy

		inv, err := s.CreateInvitation(ctx, invite, sendEmails)
		if err != nil {
			result.Failed = append(result.Failed, InviteError{
				Email: invite.Email,
				Error: err.Error(),
			})
			continue
		}

		result.Successful = append(result.Successful, InviteResult{
			Email: invite.Email,
			Role:  string(invite.Role),
			Token: inv.Token,
		})
	}

	s.logger.Info().
		Int("total", result.Total).
		Int("successful", len(result.Successful)).
		Int("failed", len(result.Failed)).
		Str("org_id", req.OrgID.String()).
		Msg("bulk invite completed")

	return result, nil
}

// ParseCSV parses a CSV file containing invite data (email,role format).
func (s *Service) ParseCSV(reader io.Reader) ([]InviteRequest, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	var invites []InviteRequest
	lineNum := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse CSV line %d: %w", lineNum+1, err)
		}
		lineNum++

		// Skip header row if present
		if lineNum == 1 && (strings.EqualFold(record[0], "email") || strings.EqualFold(record[0], "e-mail")) {
			continue
		}

		if len(record) < 1 {
			continue
		}

		email := strings.TrimSpace(record[0])
		if email == "" {
			continue
		}

		// Default role is member
		role := models.OrgRoleMember
		if len(record) >= 2 {
			roleStr := strings.TrimSpace(strings.ToLower(record[1]))
			switch roleStr {
			case "admin":
				role = models.OrgRoleAdmin
			case "member":
				role = models.OrgRoleMember
			case "readonly", "read-only", "viewer":
				role = models.OrgRoleReadonly
			case "owner":
				// Owners can't be invited, downgrade to admin
				role = models.OrgRoleAdmin
			default:
				role = models.OrgRoleMember
			}
		}

		invites = append(invites, InviteRequest{
			Email: email,
			Role:  role,
		})
	}

	if len(invites) == 0 {
		return nil, fmt.Errorf("no valid invitations found in CSV")
	}

	if len(invites) > MaxBulkInvites {
		return nil, fmt.Errorf("CSV contains %d invitations, maximum is %d", len(invites), MaxBulkInvites)
	}

	return invites, nil
}

// sendInvitationEmail sends an invitation email.
func (s *Service) sendInvitationEmail(ctx context.Context, inv *models.OrgInvitation) error {
	if s.emailService == nil {
		return nil
	}

	// Get organization details
	org, err := s.store.GetOrganizationByID(ctx, inv.OrgID)
	if err != nil {
		return fmt.Errorf("get organization: %w", err)
	}

	// Get inviter details
	inviter, err := s.store.GetUserByID(ctx, inv.InvitedBy)
	inviterName := "A team member"
	if err == nil && inviter != nil {
		inviterName = inviter.Name
		if inviterName == "" {
			inviterName = inviter.Email
		}
	}

	// Generate invite link
	inviteLink := s.GenerateInviteLink(inv.Token)

	// Send email
	data := notifications.InvitationData{
		OrgName:     org.Name,
		InviterName: inviterName,
		Role:        formatRole(inv.Role),
		InviteLink:  inviteLink,
		ExpiresAt:   inv.ExpiresAt,
		ExpiresIn:   formatDuration(time.Until(inv.ExpiresAt)),
	}

	return s.emailService.SendInvitation([]string{inv.Email}, data)
}

// ValidateInviteURL validates and extracts the token from an invite URL.
func ValidateInviteURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Extract token from path like /invite/{token}
	path := strings.TrimPrefix(parsed.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] != "invite" {
		return "", fmt.Errorf("invalid invite URL format")
	}

	token := parts[1]
	if len(token) != 64 { // 32 bytes hex encoded
		return "", fmt.Errorf("invalid token format")
	}

	return token, nil
}

// formatRole formats a role for display.
func formatRole(role models.OrgRole) string {
	switch role {
	case models.OrgRoleOwner:
		return "Owner"
	case models.OrgRoleAdmin:
		return "Administrator"
	case models.OrgRoleMember:
		return "Member"
	case models.OrgRoleReadonly:
		return "Read-Only"
	default:
		return string(role)
	}
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 1 {
		return fmt.Sprintf("%d days", days)
	} else if days == 1 {
		return "1 day"
	}

	hours := int(d.Hours())
	if hours > 1 {
		return fmt.Sprintf("%d hours", hours)
	} else if hours == 1 {
		return "1 hour"
	}

	minutes := int(d.Minutes())
	if minutes > 1 {
		return fmt.Sprintf("%d minutes", minutes)
	}

	return "less than a minute"
}
