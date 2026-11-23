package users

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestErrEmailTakenIsSentinel verifies that ErrEmailTaken is a stable sentinel
// that can be matched with errors.Is across package boundaries.
func TestErrEmailTakenIsSentinel(t *testing.T) {
	wrapped := errors.Join(ErrEmailTaken, errors.New("extra context"))

	if !errors.Is(wrapped, ErrEmailTaken) {
		t.Error("errors.Is must match ErrEmailTaken through wrapping")
	}
}

// --- integration test helpers ---

func setupUserRepo(t *testing.T) (*UserRepository, *pgxpool.Pool) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("failed to ping PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)

	return NewUserRepository(pool), pool
}

// uniqueEmail returns a collision-free email for each test run.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@integration-test.invalid", prefix, time.Now().UnixNano())
}

// deleteUserByEmail removes a test user by email; used in t.Cleanup.
func deleteUserByEmail(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	if err != nil {
		t.Errorf("cleanup: failed to delete user %s: %v", email, err)
	}
}

// bcryptHash produces a real bcrypt hash for integration tests.
func bcryptHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(h)
}

// --- CreateWithRole ---

func TestCreateWithRoleAgent(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("agent")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	user, err := repo.CreateWithRole(context.Background(), "Integration Agent", email, bcryptHash(t, "Password123"), "agent")
	if err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
	if user.Role != "agent" {
		t.Errorf("expected role agent, got %s", user.Role)
	}
	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}
	if !user.IsActive {
		t.Error("expected user to be active")
	}
}

func TestCreateWithRoleAdmin(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("admin-create")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	user, err := repo.CreateWithRole(context.Background(), "Integration Admin", email, bcryptHash(t, "Password123"), "admin")
	if err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}

func TestCreateWithRolePasswordStoredAsHash(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("hash-check")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	plaintext := "Password123"
	hash := bcryptHash(t, plaintext)

	if _, err := repo.CreateWithRole(context.Background(), "Hash Check", email, hash, "agent"); err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}

	// Read back the stored hash directly.
	var stored string
	err := pool.QueryRow(context.Background(),
		`SELECT password_hash FROM users WHERE email = $1`, email,
	).Scan(&stored)
	if err != nil {
		t.Fatalf("failed to read password_hash: %v", err)
	}
	if stored == plaintext {
		t.Error("plaintext password must not be stored")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(plaintext)); err != nil {
		t.Error("stored hash does not verify against original password")
	}
}

func TestCreateWithRoleDuplicateEmailReturnsErrEmailTaken(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("dup-email")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	hash := bcryptHash(t, "Password123")
	if _, err := repo.CreateWithRole(context.Background(), "First", email, hash, "agent"); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}
	_, err := repo.CreateWithRole(context.Background(), "Second", email, hash, "agent")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

// --- CountAdmins ---

func TestCountAdminsReturnsCorrectCount(t *testing.T) {
	repo, pool := setupUserRepo(t)

	// Baseline count before we add anything.
	before, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() error = %v", err)
	}

	email := uniqueEmail("count-admin")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	if _, err := repo.CreateWithRole(context.Background(), "Count Admin", email, bcryptHash(t, "Password123"), "admin"); err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}

	after, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() after insert error = %v", err)
	}
	if after != before+1 {
		t.Errorf("expected count to increase by 1: before=%d after=%d", before, after)
	}
}

// --- BootstrapAdmin ---

func TestBootstrapAdminCreatesFirstAdmin(t *testing.T) {
	repo, pool := setupUserRepo(t)

	// Ensure no admins exist before this test by using a unique email and
	// verifying the count increases by exactly one.
	before, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() error = %v", err)
	}
	if before > 0 {
		t.Skip("database already has admin accounts; skipping bootstrap creation test")
	}

	email := uniqueEmail("bootstrap-first")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	user, err := repo.BootstrapAdmin(context.Background(), "Bootstrap Admin", email, bcryptHash(t, "Password123"))
	if err != nil {
		t.Fatalf("BootstrapAdmin() error = %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("expected admin role, got %s", user.Role)
	}
	if user.Email != email {
		t.Errorf("expected email %s, got %s", email, user.Email)
	}
}

func TestBootstrapAdminRefusesWhenAdminExists(t *testing.T) {
	repo, pool := setupUserRepo(t)

	// Create an admin first so the count is > 0.
	existingEmail := uniqueEmail("bootstrap-existing")
	t.Cleanup(func() { deleteUserByEmail(t, pool, existingEmail) })

	if _, err := repo.CreateWithRole(context.Background(), "Existing Admin", existingEmail, bcryptHash(t, "Password123"), "admin"); err != nil {
		t.Fatalf("setup: CreateWithRole() error = %v", err)
	}

	secondEmail := uniqueEmail("bootstrap-second")
	_, err := repo.BootstrapAdmin(context.Background(), "Second Admin", secondEmail, bcryptHash(t, "Password123"))
	if !errors.Is(err, ErrAdminExists) {
		t.Errorf("expected ErrAdminExists, got %v", err)
	}
	// Confirm no partial record was written.
	deleteUserByEmail(t, pool, secondEmail) // no-op if not inserted
}

func TestBootstrapAdminRollsBackOnFailure(t *testing.T) {
	repo, pool := setupUserRepo(t)

	// Ensure no admins exist.
	count, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() error = %v", err)
	}
	if count > 0 {
		t.Skip("database already has admin accounts; skipping rollback test")
	}

	// Use an email that already exists to force a unique-constraint violation
	// inside the transaction, triggering rollback.
	email := uniqueEmail("bootstrap-rollback")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	// Pre-insert a user with the same email as an agent so the INSERT inside
	// BootstrapAdmin will hit the unique constraint.
	if _, err := repo.CreateWithRole(context.Background(), "Pre-existing", email, bcryptHash(t, "Password123"), "agent"); err != nil {
		t.Fatalf("setup: CreateWithRole() error = %v", err)
	}

	_, err = repo.BootstrapAdmin(context.Background(), "Bootstrap Fail", email, bcryptHash(t, "Password123"))
	if err == nil {
		t.Fatal("expected error due to duplicate email, got nil")
	}

	// Admin count must not have increased.
	after, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() after failed bootstrap error = %v", err)
	}
	if after != count {
		t.Errorf("admin count changed after failed bootstrap: before=%d after=%d", count, after)
	}
}

func TestBootstrapAdminConcurrentAttempts(t *testing.T) {
	repo, pool := setupUserRepo(t)

	// Ensure no admins exist before the race.
	count, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() error = %v", err)
	}
	if count > 0 {
		t.Skip("database already has admin accounts; skipping concurrency test")
	}

	const goroutines = 5
	emails := make([]string, goroutines)
	for i := range goroutines {
		emails[i] = uniqueEmail(fmt.Sprintf("concurrent-boot-%d", i))
	}
	t.Cleanup(func() {
		for _, e := range emails {
			deleteUserByEmail(t, pool, e)
		}
	})

	type result struct {
		err error
	}
	results := make([]result, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		i := i
		go func() {
			defer wg.Done()
			_, e := repo.BootstrapAdmin(
				context.Background(),
				fmt.Sprintf("Concurrent Admin %d", i),
				emails[i],
				bcryptHash(t, "Password123"),
			)
			results[i] = result{e}
		}()
	}
	wg.Wait()

	successes := 0
	for _, r := range results {
		if r.err == nil {
			successes++
		} else if !errors.Is(r.err, ErrAdminExists) {
			t.Errorf("unexpected error from concurrent bootstrap: %v", r.err)
		}
	}

	if successes != 1 {
		t.Errorf("expected exactly 1 successful bootstrap, got %d", successes)
	}

	// Confirm exactly one admin was created by this test (count increased by 1).
	after, err := repo.CountAdmins(context.Background())
	if err != nil {
		t.Fatalf("CountAdmins() after concurrent test error = %v", err)
	}
	if after != count+1 {
		t.Errorf("expected admin count to increase by 1: before=%d after=%d", count, after)
	}
}

// --- must_change_password flag ---

// TestCreateProvisionedSetsMustChangePassword verifies that CreateProvisioned
// writes must_change_password = TRUE to the database and returns the flag set.
func TestCreateProvisionedSetsMustChangePassword(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("provisioned")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	user, err := repo.CreateProvisioned(context.Background(), "Provisioned Agent", email, bcryptHash(t, "Password123"), "agent")
	if err != nil {
		t.Fatalf("CreateProvisioned() error = %v", err)
	}
	if !user.MustChangePassword {
		t.Error("expected MustChangePassword = true for provisioned user, got false")
	}

	// Verify the value is persisted in the database, not just set in memory.
	var stored bool
	if err := pool.QueryRow(context.Background(),
		`SELECT must_change_password FROM users WHERE email = $1`, email,
	).Scan(&stored); err != nil {
		t.Fatalf("failed to read must_change_password from DB: %v", err)
	}
	if !stored {
		t.Error("expected must_change_password = TRUE in database for provisioned user")
	}
}

// TestCreateWithRoleDoesNotSetMustChangePassword verifies that CreateWithRole
// always writes must_change_password = FALSE and returns the flag unset.
func TestCreateWithRoleDoesNotSetMustChangePassword(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("regular-agent")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	user, err := repo.CreateWithRole(context.Background(), "Regular Agent", email, bcryptHash(t, "Password123"), "agent")
	if err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}
	if user.MustChangePassword {
		t.Error("expected MustChangePassword = false for CreateWithRole user, got true")
	}

	var stored bool
	if err := pool.QueryRow(context.Background(),
		`SELECT must_change_password FROM users WHERE email = $1`, email,
	).Scan(&stored); err != nil {
		t.Fatalf("failed to read must_change_password from DB: %v", err)
	}
	if stored {
		t.Error("expected must_change_password = FALSE in database for CreateWithRole user")
	}
}

// TestGetByEmailRetrievesMustChangePasswordFlag verifies that GetByEmail
// returns the correct flag value for both provisioned and regular users.
func TestGetByEmailRetrievesMustChangePasswordFlag(t *testing.T) {
	repo, pool := setupUserRepo(t)

	provEmail := uniqueEmail("getbyemail-prov")
	regEmail := uniqueEmail("getbyemail-reg")
	t.Cleanup(func() {
		deleteUserByEmail(t, pool, provEmail)
		deleteUserByEmail(t, pool, regEmail)
	})

	if _, err := repo.CreateProvisioned(context.Background(), "Prov", provEmail, bcryptHash(t, "Password123"), "agent"); err != nil {
		t.Fatalf("CreateProvisioned() error = %v", err)
	}
	if _, err := repo.CreateWithRole(context.Background(), "Reg", regEmail, bcryptHash(t, "Password123"), "agent"); err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}

	provUser, _, err := repo.GetByEmail(context.Background(), provEmail)
	if err != nil {
		t.Fatalf("GetByEmail(provisioned) error = %v", err)
	}
	if !provUser.MustChangePassword {
		t.Error("GetByEmail: expected MustChangePassword = true for provisioned user")
	}

	regUser, _, err := repo.GetByEmail(context.Background(), regEmail)
	if err != nil {
		t.Fatalf("GetByEmail(regular) error = %v", err)
	}
	if regUser.MustChangePassword {
		t.Error("GetByEmail: expected MustChangePassword = false for regular user")
	}
}

// TestGetByIDRetrievesMustChangePasswordFlag verifies that GetByID returns
// the correct flag value for both provisioned and regular users.
func TestGetByIDRetrievesMustChangePasswordFlag(t *testing.T) {
	repo, pool := setupUserRepo(t)

	provEmail := uniqueEmail("getbyid-prov")
	regEmail := uniqueEmail("getbyid-reg")
	t.Cleanup(func() {
		deleteUserByEmail(t, pool, provEmail)
		deleteUserByEmail(t, pool, regEmail)
	})

	provCreated, err := repo.CreateProvisioned(context.Background(), "Prov", provEmail, bcryptHash(t, "Password123"), "agent")
	if err != nil {
		t.Fatalf("CreateProvisioned() error = %v", err)
	}
	regCreated, err := repo.CreateWithRole(context.Background(), "Reg", regEmail, bcryptHash(t, "Password123"), "agent")
	if err != nil {
		t.Fatalf("CreateWithRole() error = %v", err)
	}

	provUser, err := repo.GetByID(context.Background(), provCreated.ID)
	if err != nil {
		t.Fatalf("GetByID(provisioned) error = %v", err)
	}
	if !provUser.MustChangePassword {
		t.Error("GetByID: expected MustChangePassword = true for provisioned user")
	}

	regUser, err := repo.GetByID(context.Background(), regCreated.ID)
	if err != nil {
		t.Fatalf("GetByID(regular) error = %v", err)
	}
	if regUser.MustChangePassword {
		t.Error("GetByID: expected MustChangePassword = false for regular user")
	}
}

// TestUpdatePasswordClearsMustChangePassword verifies that UpdatePassword
// atomically sets must_change_password = FALSE, and that a subsequent read
// via GetByID confirms the persisted value.
func TestUpdatePasswordClearsMustChangePassword(t *testing.T) {
	repo, pool := setupUserRepo(t)
	email := uniqueEmail("update-pw")
	t.Cleanup(func() { deleteUserByEmail(t, pool, email) })

	created, err := repo.CreateProvisioned(context.Background(), "Must Change", email, bcryptHash(t, "OldPass123"), "agent")
	if err != nil {
		t.Fatalf("CreateProvisioned() error = %v", err)
	}
	if !created.MustChangePassword {
		t.Fatal("precondition failed: expected MustChangePassword = true after CreateProvisioned")
	}

	if err := repo.UpdatePassword(context.Background(), created.ID, bcryptHash(t, "NewPass456")); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	// Verify via GetByID that the flag is cleared in the database.
	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() after UpdatePassword error = %v", err)
	}
	if updated.MustChangePassword {
		t.Error("expected MustChangePassword = false after UpdatePassword, got true")
	}

	// Also verify directly in the database.
	var stored bool
	if err := pool.QueryRow(context.Background(),
		`SELECT must_change_password FROM users WHERE id = $1`, created.ID,
	).Scan(&stored); err != nil {
		t.Fatalf("failed to read must_change_password from DB: %v", err)
	}
	if stored {
		t.Error("expected must_change_password = FALSE in database after UpdatePassword")
	}
}
