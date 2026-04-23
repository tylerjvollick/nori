package cmd

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

var initDevCmd = &cobra.Command{
	Use:   "init-dev",
	Short: "Seed demo and test accounts into an existing database",
	Long: `Assumes 'nori init' has already been run (DB is up, personal account exists).

This command:
  1. Deletes and recreates the "Wood n Works" demo account with rich curated data
  2. Deletes and recreates the "Test Shop" test account with minimal baseline
  3. Prints test credentials to stdout for CI/Playwright

Safe to run repeatedly — always destroys and rebuilds both accounts.
Your personal account is never touched.`,
	RunE: runInitDev2,
}

func init() {
	rootCmd.AddCommand(initDevCmd)
}

// Demo account constants.
const (
	demoAccountName = "Wood n Works"
	demoEmail       = "demo@nori.dev"
	demoPassword    = "demopass123"
	demoSpaceName   = "Main Shop"
	demoSpaceSlug   = "WNW"
)

// Test account constants.
const (
	testAccountName = "Test Shop"
	testEmail       = "test@nori.dev"
	testPassword    = "testpass123"
	testSpaceName   = "Test Workshop"
	testSpaceSlug   = "TS"
)

// initDevStation defines a station for init-dev seeding.
type initDevStation struct {
	Name        string
	Description string
	WIPLimit    int
	BufferSize  int
}

var initDevDemoStations = []initDevStation{
	{Name: "Mill", Description: "Dimensioning and rough milling of lumber", WIPLimit: 3, BufferSize: 1},
	{Name: "Joinery", Description: "Mortise & tenon, dovetails, and other joints", WIPLimit: 2, BufferSize: 1},
	{Name: "Assembly", Description: "Sub-assembly and final assembly", WIPLimit: 4, BufferSize: 2},
	{Name: "Finish", Description: "Sanding, staining, and topcoat application", WIPLimit: 2, BufferSize: 1},
	{Name: "QC", Description: "Final quality check and inspection", WIPLimit: 5, BufferSize: 0},
	{Name: "Ship", Description: "Packaging, labeling, and shipping", WIPLimit: 10, BufferSize: 0},
}

var initDevTestStations = []initDevStation{
	{Name: "Cutting", Description: "Cutting and dimensioning", WIPLimit: 3, BufferSize: 1},
	{Name: "Assembly", Description: "Assembly operations", WIPLimit: 2, BufferSize: 1},
	{Name: "Finishing", Description: "Finish application", WIPLimit: 2, BufferSize: 1},
	{Name: "Quality Check", Description: "Final quality inspection", WIPLimit: 5, BufferSize: 0},
}

func runInitDev2(cmd *cobra.Command, args []string) error {
	godotenv.Load(".env")

	db, err := connectDB()
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// 1. Delete and recreate demo account.
	log.Println("=== Seeding demo account: Wood n Works ===")
	if err := deleteAccountByName(db, demoAccountName); err != nil {
		return fmt.Errorf("deleting demo account: %w", err)
	}
	if err := seedDemoAccount(db); err != nil {
		return fmt.Errorf("seeding demo account: %w", err)
	}

	// 2. Delete and recreate test account.
	log.Println()
	log.Println("=== Seeding test account: Test Shop ===")
	if err := deleteAccountByName(db, testAccountName); err != nil {
		return fmt.Errorf("deleting test account: %w", err)
	}
	if err := seedTestAccount(db); err != nil {
		return fmt.Errorf("seeding test account: %w", err)
	}

	// 3. Print credentials to stdout.
	fmt.Println()
	fmt.Println("=== Dev Accounts Ready ===")
	fmt.Println()
	fmt.Println("Demo account (rich data):")
	fmt.Printf("  NORI_DEMO_EMAIL=%s\n", demoEmail)
	fmt.Printf("  NORI_DEMO_PASSWORD=%s\n", demoPassword)
	fmt.Println()
	fmt.Println("Test account (minimal baseline):")
	fmt.Printf("  NORI_TEST_EMAIL=%s\n", testEmail)
	fmt.Printf("  NORI_TEST_PASSWORD=%s\n", testPassword)
	fmt.Println()

	return nil
}

// deleteAccountByName removes an account and all its data by name.
// No-op if the account doesn't exist.
func deleteAccountByName(db *gorm.DB, name string) error {
	var account models.Account
	if err := db.Where("name = ?", name).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("  No existing %q account found, nothing to delete.", name)
			return nil
		}
		return err
	}

	log.Printf("  Deleting existing %q account...", name)

	var spaces []models.Space
	if err := db.Where("account_id = ?", account.ID).Find(&spaces).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, space := range spaces {
			if err := deleteSpaceData(tx, space.ID); err != nil {
				return fmt.Errorf("cleaning space %s: %w", space.Name, err)
			}
			if err := tx.Exec("DELETE FROM space_member WHERE space_id = ?", space.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM space WHERE id = ?", space.ID).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM user_account WHERE account_id = ?", account.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM \"user\" WHERE default_account_id = ?", account.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM account WHERE id = ?", account.ID).Error; err != nil {
			return err
		}

		log.Printf("  Deleted %q account and all its data.", name)
		return nil
	})
}

// seedDemoAccount creates the "Wood n Works" demo account with rich curated data.
func seedDemoAccount(db *gorm.DB) error {
	// Create user.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashedStr := string(hashedPassword)
	adminRole := models.RoleAdmin

	user := &models.User{
		ID:                 uuid.New(),
		Email:              demoEmail,
		Password:           &hashedStr,
		Role:               &adminRole,
		MustChangePassword: false,
		FirstName:          initDevStrPtr("Demo"),
		LastName:           initDevStrPtr("User"),
		RecentSpaces:       models.RecentSpaces{},
	}
	if err := db.Create(user).Error; err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	// GORM skips false (zero value) on insert, so the DB default (true) wins.
	if err := db.Model(user).Update("must_change_password", false).Error; err != nil {
		return fmt.Errorf("clearing must_change_password: %w", err)
	}
	log.Printf("  Created user: %s", user.Email)

	// Create account.
	accountName := demoAccountName
	account := &models.Account{
		ID:              uuid.New(),
		Name:            &accountName,
		Plan:            models.Trial,
		CreatedByUserID: user.ID,
	}
	if err := db.Create(account).Error; err != nil {
		return fmt.Errorf("creating account: %w", err)
	}

	// Link user → account.
	user.DefaultAccountID = &account.ID
	if err := db.Model(user).Update("default_account_id", account.ID).Error; err != nil {
		return err
	}

	userAccount := &models.UserAccount{
		UserID:    user.ID,
		AccountID: account.ID,
		Role:      models.RoleAdmin,
	}
	if err := db.Create(userAccount).Error; err != nil {
		return err
	}
	log.Printf("  Created account: %s", demoAccountName)

	// Create space.
	space := &models.Space{
		ID:        uuid.New(),
		Name:      demoSpaceName,
		Slug:      demoSpaceSlug,
		AccountID: account.ID,
		IsDefault: true,
	}
	if err := db.Create(space).Error; err != nil {
		return fmt.Errorf("creating space: %w", err)
	}

	spaceMember := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	if err := db.Create(spaceMember).Error; err != nil {
		return err
	}

	user.RecentSpaces = models.RecentSpaces{space.ID}
	if err := db.Model(user).Update("recent_spaces", user.RecentSpaces).Error; err != nil {
		return err
	}
	log.Printf("  Created space: %s (%s)", space.Name, space.Slug)

	// Create stations.
	stationMap := make(map[string]*models.Station)
	for i, ds := range initDevDemoStations {
		desc := ds.Description
		station := &models.Station{
			ID:           uuid.New(),
			SpaceID:      space.ID,
			Name:         ds.Name,
			Description:  &desc,
			DisplayOrder: i + 1,
			WIPLimit:     ds.WIPLimit,
			BufferSize:   ds.BufferSize,
			IsActive:     true,
		}
		if err := db.Create(station).Error; err != nil {
			return fmt.Errorf("creating station %s: %w", ds.Name, err)
		}
		stationMap[ds.Name] = station
	}
	log.Printf("  Created %d stations", len(initDevDemoStations))

	// Create "Bistro Table" recipe (published) with 5+ steps.
	recipeRepo := repositories.NewRecipeRepository(db)
	taskRepo := repositories.NewTaskRepository(db)
	taskDepRepo := repositories.NewTaskDepRepository(db)
	timeEventRepo := repositories.NewTimeEventRepository(db)
	recipeService := services.NewRecipeService(db, recipeRepo, taskRepo, taskDepRepo)

	bistroRecipe, bistroVersion, err := createInitDevBistroRecipe(
		recipeService, db, space.ID, user.ID, stationMap,
	)
	if err != nil {
		return fmt.Errorf("creating Bistro Table recipe: %w", err)
	}
	log.Printf("  Created recipe: Bistro Table (version %d, published)", bistroVersion.VersionNumber)

	// Roll the Bistro Table job (qty 12).
	bistroJob, err := recipeService.RollRecipe(
		bistroRecipe.ID, space.ID, user.ID,
		services.RollRecipeOptions{
			Title:    initDevStrPtr("Bistro Table - Spring Batch"),
			OrderQty: initDevIntPtr(12),
		},
	)
	if err != nil {
		return fmt.Errorf("rolling Bistro Table job: %w", err)
	}
	log.Printf("  Rolled job: %s (%s, qty 12)", bistroJob.Title, bistroJob.ID)

	// Complete some tasks with realistic backdated timestamps.
	taskService := services.NewTaskService(taskRepo, taskDepRepo, timeEventRepo)
	completedCount, err := completeJobTasks(db, taskService, bistroJob.ID, user.ID, space.ID)
	if err != nil {
		return fmt.Errorf("completing job tasks: %w", err)
	}
	log.Printf("  Completed %d tasks with backdated timestamps", completedCount)

	// Create "Console Table" recipe (draft).
	if err := createInitDevConsoleRecipe(recipeService, space.ID, user.ID, stationMap); err != nil {
		return fmt.Errorf("creating Console Table recipe: %w", err)
	}
	log.Println("  Created recipe: Console Table (draft)")

	return nil
}

// createInitDevBistroRecipe creates a Bistro Table recipe with 5+ steps,
// station assignments, estimated times, batch sizes, dependencies, and publishes it.
func createInitDevBistroRecipe(
	recipeService *services.RecipeService,
	db *gorm.DB,
	spaceID uuid.UUID,
	userID uuid.UUID,
	stationMap map[string]*models.Station,
) (*models.Recipe, *models.RecipeVersion, error) {
	desc := "Solid hardwood bistro table with turned legs and breadboard ends. " +
		"Seats 2-4, designed for cafe or small kitchen use."

	recipe, _, err := recipeService.CreateRecipeWithTaskTree(
		spaceID, userID, "Bistro Table", &desc, nil,
	)
	if err != nil {
		return nil, nil, err
	}

	versions, err := listRecipeVersions(db, recipe.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(versions) == 0 {
		return nil, nil, fmt.Errorf("no versions found for recipe")
	}
	draftVersion := versions[0]
	rootTaskID := *draftVersion.RootTaskID

	type stepDef struct {
		title       string
		description string
		station     string
		estSecs     int
		batchSize   int
	}

	steps := []stepDef{
		{
			title:       "Mill lumber to dimension",
			description: "Joint, plane, rip, and crosscut all parts to final dimensions per cut list.",
			station:     "Mill",
			estSecs:     2700, // 45 min
			batchSize:   4,
		},
		{
			title:       "Cut joinery",
			description: "Cut mortise & tenon joints for leg-to-apron connections. Cut slots for breadboard ends.",
			station:     "Joinery",
			estSecs:     3600, // 60 min
			batchSize:   2,
		},
		{
			title:       "Assemble base and tabletop",
			description: "Glue tabletop panels. After cure, attach legs/aprons and breadboard ends.",
			station:     "Assembly",
			estSecs:     1800, // 30 min
			batchSize:   4,
		},
		{
			title:       "Sand and apply finish",
			description: "Progressive sand to 220 grit. Apply 3 coats of conversion varnish with scuff-sand between coats.",
			station:     "Finish",
			estSecs:     5400, // 90 min
			batchSize:   6,
		},
		{
			title:       "Final QC inspection",
			description: "Inspect all joints, finish quality, and dimensions. Apply felt pads.",
			station:     "QC",
			estSecs:     900, // 15 min
			batchSize:   1,
		},
		{
			title:       "Pack and ship",
			description: "Wrap in moving blankets, box, label, and stage for pickup/delivery.",
			station:     "Ship",
			estSecs:     600, // 10 min
			batchSize:   2,
		},
	}

	stepTasks := make([]*models.Task, len(steps))
	for i, s := range steps {
		stationID := stationMap[s.station].ID
		estSecs := s.estSecs
		batchSize := s.batchSize
		task, err := recipeService.AddRecipeStep(
			recipe.ID, rootTaskID, s.title,
			services.AddStepOptions{
				Description:       &s.description,
				StationID:         &stationID,
				EstimatedTimeSecs: &estSecs,
				BatchSize:         &batchSize,
			},
		)
		if err != nil {
			return nil, nil, fmt.Errorf("adding step %q: %w", s.title, err)
		}
		stepTasks[i] = task
	}

	// Linear dependencies: each step depends on the previous.
	for i := 1; i < len(stepTasks); i++ {
		if err := recipeService.AddStepDependency(
			recipe.ID,
			stepTasks[i].ID,
			stepTasks[i-1].ID,
		); err != nil {
			return nil, nil, fmt.Errorf("adding dependency %d→%d: %w", i, i-1, err)
		}
	}

	// Publish the recipe.
	published, err := recipeService.PublishVersion(recipe.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("publishing: %w", err)
	}

	var updatedRecipe models.Recipe
	if err := db.First(&updatedRecipe, "id = ?", recipe.ID).Error; err != nil {
		return nil, nil, err
	}

	return &updatedRecipe, published, nil
}

// createInitDevConsoleRecipe creates a Console Table recipe in draft state.
func createInitDevConsoleRecipe(
	recipeService *services.RecipeService,
	spaceID uuid.UUID,
	userID uuid.UUID,
	stationMap map[string]*models.Station,
) error {
	desc := "Narrow console table with tapered legs and a single drawer. " +
		"Perfect for entryways and hallways."

	recipe, version, err := recipeService.CreateRecipeWithTaskTree(
		spaceID, userID, "Console Table", &desc, nil,
	)
	if err != nil {
		return err
	}
	rootTaskID := *version.RootTaskID

	type stepDef struct {
		title   string
		station string
		estSecs int
	}
	steps := []stepDef{
		{title: "Mill legs and rails", station: "Mill", estSecs: 1800},
		{title: "Cut mortise & tenon joints", station: "Joinery", estSecs: 2400},
		{title: "Build drawer box", station: "Joinery", estSecs: 1800},
		{title: "Assemble carcase", station: "Assembly", estSecs: 1200},
		{title: "Fit drawer and hardware", station: "Joinery", estSecs: 900},
		{title: "Sand and apply finish", station: "Finish", estSecs: 3600},
	}

	for _, s := range steps {
		stationID := stationMap[s.station].ID
		estSecs := s.estSecs
		_, err := recipeService.AddRecipeStep(
			recipe.ID, rootTaskID, s.title,
			services.AddStepOptions{
				StationID:         &stationID,
				EstimatedTimeSecs: &estSecs,
			},
		)
		if err != nil {
			return fmt.Errorf("adding step %q: %w", s.title, err)
		}
	}

	_ = recipe // keep reference clear
	return nil
}

// seedTestAccount creates the "Test Shop" test account with minimal baseline.
func seedTestAccount(db *gorm.DB) error {
	// Create user.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashedStr := string(hashedPassword)
	adminRole := models.RoleAdmin

	user := &models.User{
		ID:                 uuid.New(),
		Email:              testEmail,
		Password:           &hashedStr,
		Role:               &adminRole,
		MustChangePassword: false,
		FirstName:          initDevStrPtr("Test"),
		LastName:           initDevStrPtr("Admin"),
		RecentSpaces:       models.RecentSpaces{},
	}
	if err := db.Create(user).Error; err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	if err := db.Model(user).Update("must_change_password", false).Error; err != nil {
		return fmt.Errorf("clearing must_change_password: %w", err)
	}
	log.Printf("  Created user: %s", user.Email)

	// Create account.
	accountName := testAccountName
	account := &models.Account{
		ID:              uuid.New(),
		Name:            &accountName,
		Plan:            models.Trial,
		CreatedByUserID: user.ID,
	}
	if err := db.Create(account).Error; err != nil {
		return fmt.Errorf("creating account: %w", err)
	}

	// Link user → account.
	user.DefaultAccountID = &account.ID
	if err := db.Model(user).Update("default_account_id", account.ID).Error; err != nil {
		return err
	}

	userAccount := &models.UserAccount{
		UserID:    user.ID,
		AccountID: account.ID,
		Role:      models.RoleAdmin,
	}
	if err := db.Create(userAccount).Error; err != nil {
		return err
	}
	log.Printf("  Created account: %s", testAccountName)

	// Create space.
	space := &models.Space{
		ID:        uuid.New(),
		Name:      testSpaceName,
		Slug:      testSpaceSlug,
		AccountID: account.ID,
		IsDefault: true,
	}
	if err := db.Create(space).Error; err != nil {
		return fmt.Errorf("creating space: %w", err)
	}

	spaceMember := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	if err := db.Create(spaceMember).Error; err != nil {
		return err
	}

	user.RecentSpaces = models.RecentSpaces{space.ID}
	if err := db.Model(user).Update("recent_spaces", user.RecentSpaces).Error; err != nil {
		return err
	}
	log.Printf("  Created space: %s (%s)", space.Name, space.Slug)

	// Create default stations.
	for i, ds := range initDevTestStations {
		desc := ds.Description
		station := &models.Station{
			ID:           uuid.New(),
			SpaceID:      space.ID,
			Name:         ds.Name,
			Description:  &desc,
			DisplayOrder: i + 1,
			WIPLimit:     ds.WIPLimit,
			BufferSize:   ds.BufferSize,
			IsActive:     true,
		}
		if err := db.Create(station).Error; err != nil {
			return fmt.Errorf("creating station %s: %w", ds.Name, err)
		}
	}
	log.Printf("  Created %d stations", len(initDevTestStations))

	return nil
}

// initDevStrPtr returns a pointer to a string.
func initDevStrPtr(s string) *string { return &s }

// initDevIntPtr returns a pointer to an int.
func initDevIntPtr(i int) *int { return &i }
