package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors for team operations.
var (
	ErrTeamNotFound      = errors.New("team not found")
	ErrTeamNameTaken     = errors.New("team name is already taken")
	ErrMembershipExists  = errors.New("user is already a member of this team")
	ErrMembershipNotFound = errors.New("membership not found")
	ErrUserNotFound      = errors.New("user not found")
)

// TeamRepository handles database operations for teams.
type TeamRepository struct {
	db *pgxpool.Pool
}

// NewTeamRepository creates a team repository.
func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

// CreateTeam inserts a new team.
func (r *TeamRepository) CreateTeam(
	ctx context.Context,
	name string,
	description *string,
) (*models.Team, error) {
	var team models.Team
	err := r.db.QueryRow(ctx,
		`INSERT INTO teams (name, description)
		 VALUES ($1, $2)
		 RETURNING id, name, description, created_at`,
		name, description,
	).Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrTeamNameTaken
		}
		return nil, fmt.Errorf("failed to create team: %w", err)
	}
	return &team, nil
}

// ListTeams returns all teams ordered by name.
func (r *TeamRepository) ListTeams(ctx context.Context) ([]models.Team, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, created_at FROM teams ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer rows.Close()

	teams := []models.Team{}
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate teams: %w", err)
	}
	return teams, nil
}

// GetTeamByID retrieves a single team by UUID.
func (r *TeamRepository) GetTeamByID(ctx context.Context, id string) (*models.Team, error) {
	var team models.Team
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, created_at FROM teams WHERE id = $1::uuid`,
		id,
	).Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, fmt.Errorf("failed to retrieve team: %w", err)
	}
	return &team, nil
}

// UpdateTeamRequest carries the fields to update. A nil pointer means "leave unchanged".
// SetDescription=true with Description=nil clears the description.
type UpdateTeamRequest struct {
	Name            *string
	Description     *string
	SetDescription  bool
}

// UpdateTeam applies a partial update to a team.
func (r *TeamRepository) UpdateTeam(
	ctx context.Context,
	id string,
	req UpdateTeamRequest,
) (*models.Team, error) {
	var team models.Team
	err := r.db.QueryRow(ctx,
		`UPDATE teams
		 SET name        = CASE WHEN $2 THEN $3::varchar ELSE name        END,
		     description = CASE WHEN $4 THEN $5::text    ELSE description END
		 WHERE id = $1::uuid
		 RETURNING id, name, description, created_at`,
		id,
		req.Name != nil, stringPtrVal(req.Name),
		req.SetDescription, stringPtrVal(req.Description),
	).Scan(&team.ID, &team.Name, &team.Description, &team.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrTeamNameTaken
		}
		return nil, fmt.Errorf("failed to update team: %w", err)
	}
	return &team, nil
}

// AddTeamMember adds a user to a team with the given role.
func (r *TeamRepository) AddTeamMember(
	ctx context.Context,
	teamID string,
	userID string,
	teamRole string,
) (*models.TeamMember, error) {
	if _, err := r.GetTeamByID(ctx, teamID); err != nil {
		return nil, err
	}

	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1::uuid)`, userID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("failed to verify user: %w", err)
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	var m models.TeamMember
	err := r.db.QueryRow(ctx,
		`INSERT INTO team_members (team_id, user_id, team_role)
		 VALUES ($1::uuid, $2::uuid, $3)
		 RETURNING team_id, user_id, team_role, joined_at`,
		teamID, userID, teamRole,
	).Scan(&m.TeamID, &m.UserID, &m.TeamRole, &m.JoinedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrMembershipExists
		}
		return nil, fmt.Errorf("failed to add team member: %w", err)
	}
	return &m, nil
}

// RemoveTeamMember removes a user from a team.
func (r *TeamRepository) RemoveTeamMember(
	ctx context.Context,
	teamID string,
	userID string,
) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM team_members WHERE team_id = $1::uuid AND user_id = $2::uuid`,
		teamID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMembershipNotFound
	}
	return nil
}

// ListTeamMembers returns all members of a team with their user details.
func (r *TeamRepository) ListTeamMembers(
	ctx context.Context,
	teamID string,
) ([]models.TeamMember, error) {
	if _, err := r.GetTeamByID(ctx, teamID); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT tm.team_id, tm.user_id, tm.team_role, tm.joined_at,
		        u.id, u.full_name, u.email, u.role, u.is_active, u.created_at, u.updated_at
		 FROM team_members tm
		 JOIN users u ON u.id = tm.user_id
		 WHERE tm.team_id = $1::uuid
		 ORDER BY tm.joined_at ASC`,
		teamID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}
	defer rows.Close()

	members := []models.TeamMember{}
	for rows.Next() {
		var m models.TeamMember
		var u models.User
		if err := rows.Scan(
			&m.TeamID, &m.UserID, &m.TeamRole, &m.JoinedAt,
			&u.ID, &u.FullName, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		m.User = &u
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate team members: %w", err)
	}
	return members, nil
}

func stringPtrVal(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
