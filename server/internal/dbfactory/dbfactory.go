// Package dbfactory provides test data builder functions that insert real
// records into PostgreSQL via GORM transactions. Each builder handles required
// foreign keys automatically: if a dependency is not provided via overrides,
// the builder creates one. Use with dbtest.TestTx() for per-test isolation.
package dbfactory

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/models"
)

// --- Account & User ---

// Account creates an Account record. If CreatedByUserID is zero, a User is
// created first and linked.
func Account(tx *gorm.DB, overrides ...func(*models.Account)) models.Account {
	m := models.Account{
		ID:   uuid.New(),
		Plan: models.Trial,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.CreatedByUserID == uuid.Nil {
		u := User(tx)
		m.CreatedByUserID = u.ID
	}
	if m.Name == nil {
		n := "Test Account"
		m.Name = &n
	}
	must(tx.Create(&m).Error)
	return m
}

// User creates a User record with sensible defaults.
func User(tx *gorm.DB, overrides ...func(*models.User)) models.User {
	m := models.User{
		ID:                 uuid.New(),
		Email:              fmt.Sprintf("user-%s@test.nori.dev", uuid.New().String()[:8]),
		MustChangePassword: false,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	must(tx.Create(&m).Error)
	return m
}

// UserAccount creates a UserAccount join record. If UserID or AccountID are
// zero, the corresponding records are created.
func UserAccount(tx *gorm.DB, overrides ...func(*models.UserAccount)) models.UserAccount {
	m := models.UserAccount{
		ID:   uuid.New(),
		Role: models.RoleUser,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.UserID == uuid.Nil {
		u := User(tx)
		m.UserID = u.ID
	}
	if m.AccountID == uuid.Nil {
		a := Account(tx, func(a *models.Account) { a.CreatedByUserID = m.UserID })
		m.AccountID = a.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- Space ---

// Space creates a Space record. If AccountID is zero, an Account (and its
// User) are created.
func Space(tx *gorm.DB, overrides ...func(*models.Space)) models.Space {
	m := models.Space{
		ID:   uuid.New(),
		Name: "Test Space",
		Slug: uuid.New().String()[:8],
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.AccountID == uuid.Nil {
		a := Account(tx)
		m.AccountID = a.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- Station ---

// Station creates a Station record. If SpaceID is zero, a Space (and its
// Account chain) is created.
func Station(tx *gorm.DB, overrides ...func(*models.Station)) models.Station {
	m := models.Station{
		ID:       uuid.New(),
		Name:     "Test Station",
		WIPLimit: 1,
		IsActive: true,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.SpaceID == uuid.Nil {
		s := Space(tx)
		m.SpaceID = s.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- Task ---

// Task creates a Task record with a generated hierarchical ID. If SpaceID
// or CreatedByID are zero, the corresponding records are created.
func Task(tx *gorm.DB, overrides ...func(*models.Task)) models.Task {
	m := models.Task{
		ID:     fmt.Sprintf("test-%s", uuid.New().String()[:8]),
		Title:  "Test Task",
		Type:   models.TaskTypeTask,
		Status: models.TaskStatusOpen,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.SpaceID == uuid.Nil {
		s := Space(tx)
		m.SpaceID = s.ID
	}
	if m.CreatedByID == uuid.Nil {
		u := User(tx)
		m.CreatedByID = u.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- Recipe & RecipeVersion ---

// Recipe creates a Recipe record. If SpaceID or CreatedByID are zero, the
// corresponding records are created.
func Recipe(tx *gorm.DB, overrides ...func(*models.Recipe)) models.Recipe {
	m := models.Recipe{
		ID:       uuid.New(),
		Slug:     fmt.Sprintf("test-recipe-%s", uuid.New().String()[:8]),
		IsActive: true,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.SpaceID == uuid.Nil {
		s := Space(tx)
		m.SpaceID = s.ID
	}
	if m.CreatedByID == uuid.Nil {
		u := User(tx)
		m.CreatedByID = u.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// RecipeVersion creates a RecipeVersion record. If RecipeID is zero, a Recipe
// (and its dependencies) is created. If AuthorID is zero, a User is created.
func RecipeVersion(tx *gorm.DB, overrides ...func(*models.RecipeVersion)) models.RecipeVersion {
	m := models.RecipeVersion{
		VersionNumber: 1,
		Status:        models.RecipeVersionStatusDraft,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.RecipeID == uuid.Nil {
		r := Recipe(tx)
		m.RecipeID = r.ID
	}
	if m.AuthorID == uuid.Nil {
		u := User(tx)
		m.AuthorID = u.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- TimeEvent ---

// TimeEvent creates a TimeEvent record. If SpaceID or UserID are zero, the
// corresponding records are created.
func TimeEvent(tx *gorm.DB, overrides ...func(*models.TimeEvent)) models.TimeEvent {
	m := models.TimeEvent{
		ID:        uuid.New(),
		EventType: models.TimeEventTypeCheckIn,
		Source:    models.TimeEventSourceManual,
		Timestamp: time.Now(),
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.SpaceID == uuid.Nil {
		s := Space(tx)
		m.SpaceID = s.ID
	}
	if m.UserID == uuid.Nil {
		u := User(tx)
		m.UserID = u.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- Product & ProductVariant ---

// Product creates a Product record. If SpaceID is zero, a Space (and its
// Account chain) is created.
func Product(tx *gorm.DB, overrides ...func(*models.Product)) models.Product {
	m := models.Product{
		ID:       uuid.New(),
		Name:     fmt.Sprintf("Test Product %s", uuid.New().String()[:8]),
		IsActive: true,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.SpaceID == uuid.Nil {
		s := Space(tx)
		m.SpaceID = s.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// ProductVariant creates a ProductVariant record. If ProductID is zero, a
// Product (and its Space chain) is created.
func ProductVariant(tx *gorm.DB, overrides ...func(*models.ProductVariant)) models.ProductVariant {
	m := models.ProductVariant{
		ID:       uuid.New(),
		Name:     fmt.Sprintf("Test Variant %s", uuid.New().String()[:8]),
		IsActive: true,
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.ProductID == uuid.Nil {
		p := Product(tx)
		m.ProductID = p.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// --- CostEntry ---

// CostEntry creates a CostEntry record. If TaskID is empty, a Task is
// created. If CreatedByID is zero, a User is created.
func CostEntry(tx *gorm.DB, overrides ...func(*models.CostEntry)) models.CostEntry {
	m := models.CostEntry{
		ID:          uuid.New(),
		CostType:    models.CostTypeLabor,
		Description: "Test cost entry",
		Amount:      decimal.NewFromFloat(10.00),
	}
	for _, fn := range overrides {
		fn(&m)
	}
	if m.TaskID == "" {
		t := Task(tx)
		m.TaskID = t.ID
	}
	if m.CreatedByID == uuid.Nil {
		u := User(tx)
		m.CreatedByID = u.ID
	}
	must(tx.Create(&m).Error)
	return m
}

// must panics on error. Builders are test helpers; panicking on DB errors
// produces clear test failures with stack traces.
func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("dbfactory: %v", err))
	}
}
