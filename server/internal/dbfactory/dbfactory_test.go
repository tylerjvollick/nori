package dbfactory_test

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestMain(m *testing.M) {
	// Go test runs from the package directory (server/internal/dbfactory/).
	// Migrations are at server/migrations/, two levels up.
	os.Exit(dbtest.SetupTestMain(m, os.DirFS("../..")))
}

func TestUser(t *testing.T) {
	tx := dbtest.TestTx(t)
	u := dbfactory.User(tx)

	if u.ID == uuid.Nil {
		t.Fatal("expected non-nil user ID")
	}
	if u.Email == "" {
		t.Fatal("expected non-empty email")
	}

	var count int64
	tx.Model(&models.User{}).Where("id = ?", u.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 user row, got %d", count)
	}
}

func TestUser_WithOverrides(t *testing.T) {
	tx := dbtest.TestTx(t)
	first := "Alice"
	u := dbfactory.User(tx, func(u *models.User) {
		u.Email = "alice@test.nori.dev"
		u.FirstName = &first
	})

	if u.Email != "alice@test.nori.dev" {
		t.Fatalf("expected overridden email, got %s", u.Email)
	}
	if u.FirstName == nil || *u.FirstName != "Alice" {
		t.Fatal("expected overridden first name")
	}
}

func TestAccount(t *testing.T) {
	tx := dbtest.TestTx(t)
	a := dbfactory.Account(tx)

	if a.ID == uuid.Nil {
		t.Fatal("expected non-nil account ID")
	}
	if a.CreatedByUserID == uuid.Nil {
		t.Fatal("expected auto-created user")
	}

	// Verify the user was actually inserted
	var count int64
	tx.Model(&models.User{}).Where("id = ?", a.CreatedByUserID).Count(&count)
	if count != 1 {
		t.Fatal("expected auto-created user to exist in DB")
	}
}

func TestSpace(t *testing.T) {
	tx := dbtest.TestTx(t)
	s := dbfactory.Space(tx)

	if s.ID == uuid.Nil {
		t.Fatal("expected non-nil space ID")
	}
	if s.AccountID == uuid.Nil {
		t.Fatal("expected auto-created account")
	}
}

func TestStation(t *testing.T) {
	tx := dbtest.TestTx(t)
	st := dbfactory.Station(tx)

	if st.ID == uuid.Nil {
		t.Fatal("expected non-nil station ID")
	}
	if st.SpaceID == uuid.Nil {
		t.Fatal("expected auto-created space")
	}
}

func TestStation_WithExistingSpace(t *testing.T) {
	tx := dbtest.TestTx(t)
	s := dbfactory.Space(tx)
	st := dbfactory.Station(tx, func(st *models.Station) {
		st.SpaceID = s.ID
		st.Name = "Custom Station"
	})

	if st.SpaceID != s.ID {
		t.Fatal("expected station to use provided space")
	}
	if st.Name != "Custom Station" {
		t.Fatalf("expected overridden name, got %s", st.Name)
	}
}

func TestTask(t *testing.T) {
	tx := dbtest.TestTx(t)
	task := dbfactory.Task(tx)

	if task.ID == "" {
		t.Fatal("expected non-empty task ID")
	}
	if task.SpaceID == uuid.Nil {
		t.Fatal("expected auto-created space")
	}
	if task.CreatedByID == uuid.Nil {
		t.Fatal("expected auto-created user")
	}
}

func TestRecipe(t *testing.T) {
	tx := dbtest.TestTx(t)
	r := dbfactory.Recipe(tx)

	if r.ID == uuid.Nil {
		t.Fatal("expected non-nil recipe ID")
	}
	if r.SpaceID == uuid.Nil {
		t.Fatal("expected auto-created space")
	}
	if r.CreatedByID == uuid.Nil {
		t.Fatal("expected auto-created user")
	}
}

func TestRecipeVersion(t *testing.T) {
	tx := dbtest.TestTx(t)
	rv := dbfactory.RecipeVersion(tx)

	if rv.ID == 0 {
		t.Fatal("expected non-zero recipe version ID")
	}
	if rv.RecipeID == uuid.Nil {
		t.Fatal("expected auto-created recipe")
	}
	if rv.AuthorID == uuid.Nil {
		t.Fatal("expected auto-created author")
	}
}

func TestTimeEvent(t *testing.T) {
	tx := dbtest.TestTx(t)
	te := dbfactory.TimeEvent(tx)

	if te.ID == uuid.Nil {
		t.Fatal("expected non-nil time event ID")
	}
	if te.SpaceID == uuid.Nil {
		t.Fatal("expected auto-created space")
	}
	if te.UserID == uuid.Nil {
		t.Fatal("expected auto-created user")
	}
}

func TestCostEntry(t *testing.T) {
	tx := dbtest.TestTx(t)
	ce := dbfactory.CostEntry(tx)

	if ce.ID == uuid.Nil {
		t.Fatal("expected non-nil cost entry ID")
	}
	if ce.TaskID == "" {
		t.Fatal("expected auto-created task")
	}
	if ce.CreatedByID == uuid.Nil {
		t.Fatal("expected auto-created user")
	}
	if !ce.Amount.Equal(decimal.NewFromFloat(10.00)) {
		t.Fatalf("expected default amount 10.00, got %s", ce.Amount.String())
	}
}

func TestUserAccount(t *testing.T) {
	tx := dbtest.TestTx(t)
	ua := dbfactory.UserAccount(tx)

	if ua.ID == uuid.Nil {
		t.Fatal("expected non-nil user account ID")
	}
	if ua.UserID == uuid.Nil {
		t.Fatal("expected auto-created user")
	}
	if ua.AccountID == uuid.Nil {
		t.Fatal("expected auto-created account")
	}
}
