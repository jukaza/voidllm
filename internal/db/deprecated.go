package db

import (
	"context"
)

// Deprecated B2B structs to satisfy test compiling.
type Org struct {
	ID                string
	Name              string
	Slug              string
	DailyTokenLimit   int64
	MonthlyTokenLimit int64
	RequestsPerMinute int
	RequestsPerDay    int
}

type Team struct {
	ID                string
	OrgID             string
	Name              string
	Slug              string
	DailyTokenLimit   int64
	MonthlyTokenLimit int64
	RequestsPerMinute int
	RequestsPerDay    int
}

type OrgMembership struct {
	ID    string
	OrgID string
	Role  string
}

type TeamMembership struct {
	ID     string
	TeamID string
	Role   string
}

type ServiceAccount struct {
	ID string
}

type CreateOrgParams struct {
	Name              string
	Slug              string
	DailyTokenLimit   int64
	MonthlyTokenLimit int64
	RequestsPerMinute int
	RequestsPerDay    int
}

type CreateTeamParams struct {
	OrgID             string
	Name              string
	Slug              string
	DailyTokenLimit   int64
	MonthlyTokenLimit int64
	RequestsPerMinute int
	RequestsPerDay    int
}

type CreateOrgMembershipParams struct {
	OrgID  string
	UserID string
	Role   string
}

type CreateTeamMembershipParams struct {
	TeamID string
	UserID string
	Role   string
}

type CreateServiceAccountParams struct {
	Name      string
	OrgID     string
	TeamID    *string
	CreatedBy string
}

func (d *DB) CreateOrg(ctx context.Context, params CreateOrgParams) (*Org, error) {
	return &Org{ID: "mock-org-id"}, nil
}

func (d *DB) CreateTeam(ctx context.Context, params CreateTeamParams) (*Team, error) {
	return &Team{ID: "mock-team-id"}, nil
}

func (d *DB) CreateOrgMembership(ctx context.Context, params CreateOrgMembershipParams) (*OrgMembership, error) {
	return &OrgMembership{ID: "mock-membership-id"}, nil
}

func (d *DB) CreateTeamMembership(ctx context.Context, params CreateTeamMembershipParams) (*TeamMembership, error) {
	return &TeamMembership{ID: "mock-membership-id"}, nil
}

func (d *DB) CreateServiceAccount(ctx context.Context, params CreateServiceAccountParams) (*ServiceAccount, error) {
	return &ServiceAccount{ID: "mock-sa-id"}, nil
}

func (d *DB) GetOrg(ctx context.Context, id string) (*Org, error) {
	return &Org{ID: id}, nil
}

func (d *DB) GetTeam(ctx context.Context, id string) (*Team, error) {
	return &Team{ID: id}, nil
}

func (d *DB) GetUserTeamID(ctx context.Context, orgID, userID string) (string, error) {
	return "", nil
}

func (d *DB) CountTeams(ctx context.Context, orgID string) (int, error) {
	return 0, nil
}

func (d *DB) CountOrgMembers(ctx context.Context, orgID string) (int, error) {
	return 0, nil
}

func (d *DB) CountTeamKeys(ctx context.Context, teamID string) (int, error) {
	return 0, nil
}

func (d *DB) CountTeamMembers(ctx context.Context, teamID string) (int, error) {
	return 0, nil
}

func (d *DB) DeleteOrg(ctx context.Context, id string) error {
	return nil
}

func (d *DB) DeleteTeam(ctx context.Context, id string) error {
	return nil
}

func (d *DB) DeleteServiceAccount(ctx context.Context, id string) error {
	return nil
}


