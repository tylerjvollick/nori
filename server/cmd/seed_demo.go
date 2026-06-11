package cmd

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

var resetFlag bool

var seedDemoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Seed a curated demo environment",
	Long: `Create a "Vollick Woodworks" demo account with realistic data:
  - Workshop stations with WIP limits
  - "Bistro Table" recipe (published) with 5 steps
  - A rolled job (qty 12) with partial completion
  - "Console Table" recipe (draft)

Use --reset to wipe and rebuild the demo data.`,
	RunE: runSeedDemo,
}

func init() {
	seedDemoCmd.Flags().BoolVar(&resetFlag, "reset", false, "Wipe existing demo data before seeding")
	seedCmd.AddCommand(seedDemoCmd)
}

// demoStation defines a station with display metadata.
type demoStation struct {
	Name        string
	Description string
	WIPLimit    int
	BufferSize  int
}

var demoStations = []demoStation{
	{Name: "Mill", Description: "Dimensioning and rough milling of lumber", WIPLimit: 2, BufferSize: 1},
	{Name: "Joinery", Description: "Mortise & tenon, dovetails, and other joints", WIPLimit: 2, BufferSize: 1},
	{Name: "Glue-Up", Description: "Panel glue-ups and sub-assembly bonding", WIPLimit: 3, BufferSize: 2},
	{Name: "Finish", Description: "Sanding, staining, and topcoat application", WIPLimit: 2, BufferSize: 1},
	{Name: "QC", Description: "Final quality check and packaging", WIPLimit: 5, BufferSize: 0},
}

func runSeedDemo(cmd *cobra.Command, args []string) error {
	godotenv.Load(".env")

	db, err := connectDB()
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	if resetFlag {
		log.Println("Resetting demo data...")
		if err := resetDemoData(db); err != nil {
			return fmt.Errorf("reset failed: %w", err)
		}
		log.Println("Demo data reset complete.")
	}

	log.Println("Seeding demo environment...")
	if err := seedDemo(db); err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	log.Println("Demo environment seeded successfully.")
	return nil
}

// connectDB opens a GORM connection using DB_* environment variables.
func connectDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		envOrDefault("DB_HOST", "localhost"),
		envOrDefault("DB_USER", "postgres"),
		os.Getenv("DB_PASSWORD"),
		envOrDefault("DB_NAME", "nori"),
		envOrDefault("DB_PORT", "5433"),
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// resetDemoData deletes all data associated with the "Vollick Woodworks" account.
func resetDemoData(db *gorm.DB) error {
	// Find the demo account by name.
	var account models.Account
	demoName := "Vollick Woodworks"
	if err := db.Where("name = ?", demoName).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Println("No demo account found, nothing to reset.")
			return nil
		}
		return err
	}

	// Find all spaces under this account.
	var spaces []models.Space
	if err := db.Where("account_id = ?", account.ID).Find(&spaces).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, space := range spaces {
			if err := deleteSpaceData(tx, space.ID); err != nil {
				return fmt.Errorf("cleaning space %s: %w", space.Name, err)
			}
			// Delete space members.
			if err := tx.Exec("DELETE FROM space_member WHERE space_id = ?", space.ID).Error; err != nil {
				return err
			}
			// Delete the space itself.
			if err := tx.Exec("DELETE FROM space WHERE id = ?", space.ID).Error; err != nil {
				return err
			}
		}

		// Delete user-account relationships.
		if err := tx.Exec("DELETE FROM user_account WHERE account_id = ?", account.ID).Error; err != nil {
			return err
		}

		// Delete users whose default_account_id is this account.
		if err := tx.Exec("DELETE FROM \"user\" WHERE default_account_id = ?", account.ID).Error; err != nil {
			return err
		}

		// Delete the account.
		if err := tx.Exec("DELETE FROM account WHERE id = ?", account.ID).Error; err != nil {
			return err
		}

		return nil
	})
}

// deleteSpaceData removes all work data within a space (same pattern as test handler reset).
func deleteSpaceData(tx *gorm.DB, spaceID uuid.UUID) error {
	queries := []string{
		"DELETE FROM product_variant WHERE product_id IN (SELECT id FROM product WHERE space_id = $1)",
		"DELETE FROM product WHERE space_id = $1",
		"DELETE FROM cost_entry WHERE task_id IN (SELECT id FROM task WHERE space_id = $1)",
		"DELETE FROM activity_entry WHERE task_id IN (SELECT id FROM task WHERE space_id = $1)",
		"DELETE FROM bom_item WHERE recipe_version_id IN (SELECT id FROM recipe_version WHERE recipe_id IN (SELECT id FROM recipe WHERE space_id = $1))",
		"DELETE FROM time_event WHERE space_id = $1",
		"DELETE FROM task_tag WHERE task_id IN (SELECT id FROM task WHERE space_id = $1)",
		"DELETE FROM task_dep WHERE from_task_id IN (SELECT id FROM task WHERE space_id = $1) OR to_task_id IN (SELECT id FROM task WHERE space_id = $1)",
		"UPDATE recipe SET current_version_id = NULL WHERE space_id = $1",
		"DELETE FROM recipe_version WHERE recipe_id IN (SELECT id FROM recipe WHERE space_id = $1)",
		"DELETE FROM recipe_tag WHERE recipe_id IN (SELECT id FROM recipe WHERE space_id = $1)",
		"DELETE FROM recipe WHERE space_id = $1",
		"UPDATE task SET parent_id = NULL WHERE space_id = $1",
		"DELETE FROM task WHERE space_id = $1",
		"DELETE FROM station WHERE space_id = $1",
	}
	for _, q := range queries {
		if err := tx.Exec(q, spaceID).Error; err != nil {
			return fmt.Errorf("executing %q: %w", q[:40], err)
		}
	}
	return nil
}

// seedDemo creates the full demo environment.
func seedDemo(db *gorm.DB) error {
	// Check if demo account already exists.
	var count int64
	demoName := "Vollick Woodworks"
	db.Model(&models.Account{}).Where("name = ?", demoName).Count(&count)
	if count > 0 {
		return fmt.Errorf("demo account %q already exists (use --reset to rebuild)", demoName)
	}

	// 1. Create admin user.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashedStr := string(hashedPassword)
	adminRole := models.RoleAdmin
	user := &models.User{
		ID:                 uuid.New(),
		Email:              "tyler@vollickwoodworks.com",
		Password:           &hashedStr,
		Role:               &adminRole,
		MustChangePassword: false,
		FirstName:          demoStrPtr("Tyler"),
		LastName:           demoStrPtr("Vollick"),
		RecentSpaces:       models.RecentSpaces{},
	}
	if err := db.Create(user).Error; err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	log.Printf("  Created user: %s", user.Email)

	// 2. Create account.
	account := &models.Account{
		ID:              uuid.New(),
		Name:            &demoName,
		Plan:            models.Trial,
		CreatedByUserID: user.ID,
	}
	if err := db.Create(account).Error; err != nil {
		return fmt.Errorf("creating account: %w", err)
	}

	// Link user → account.
	user.DefaultAccountID = &account.ID
	if err := db.Model(user).Update("default_account_id", account.ID).Error; err != nil {
		return fmt.Errorf("updating user default account: %w", err)
	}

	userAccount := &models.UserAccount{
		UserID:    user.ID,
		AccountID: account.ID,
		Role:      models.RoleAdmin,
	}
	if err := db.Create(userAccount).Error; err != nil {
		return fmt.Errorf("creating user-account: %w", err)
	}
	log.Printf("  Created account: %s", demoName)

	// 3. Create space.
	space := &models.Space{
		ID:        uuid.New(),
		Name:      "Workshop",
		Slug:      "VW",
		AccountID: account.ID,
		IsDefault: true,
	}
	if err := db.Create(space).Error; err != nil {
		return fmt.Errorf("creating space: %w", err)
	}

	// Add user to space.
	spaceMember := &models.SpaceMember{
		UserID:  user.ID,
		SpaceID: space.ID,
	}
	if err := db.Create(spaceMember).Error; err != nil {
		return fmt.Errorf("creating space member: %w", err)
	}

	// Update user's recent spaces.
	user.RecentSpaces = models.RecentSpaces{space.ID}
	if err := db.Model(user).Update("recent_spaces", user.RecentSpaces).Error; err != nil {
		return fmt.Errorf("updating recent spaces: %w", err)
	}
	log.Printf("  Created space: %s (%s)", space.Name, space.Slug)

	// 4. Create stations.
	stationMap := make(map[string]*models.Station)
	for i, ds := range demoStations {
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
	log.Printf("  Created %d stations", len(demoStations))

	// 5. Create "Bistro Table" recipe with task tree.
	recipeRepo := repositories.NewRecipeRepository(db)
	taskRepo := repositories.NewTaskRepository(db)
	taskDepRepo := repositories.NewTaskDepRepository(db)
	timeEventRepo := repositories.NewTimeEventRepository(db)
	recipeService := services.NewRecipeService(db, recipeRepo, taskRepo, taskDepRepo)

	bistroRecipe, bistroVersion, err := createBistroTableRecipe(
		recipeService, db, space.ID, user.ID, stationMap,
	)
	if err != nil {
		return fmt.Errorf("creating Bistro Table recipe: %w", err)
	}
	log.Printf("  Created recipe: Bistro Table (version %d, published)", bistroVersion.VersionNumber)

	// 6. Roll the Bistro Table job (qty 12).
	bistroJob, err := recipeService.RollRecipe(
		bistroRecipe.ID, space.ID, user.ID,
		services.RollRecipeOptions{
			Title:    demoStrPtr("Bistro Table - Spring Batch"),
			OrderQty: demoIntPtr(12),
		},
	)
	if err != nil {
		return fmt.Errorf("rolling Bistro Table job: %w", err)
	}
	log.Printf("  Rolled job: %s (%s, qty 12)", bistroJob.Title, bistroJob.ID)

	// 7. Complete some tasks in the job with realistic timestamps.
	taskService := services.NewTaskService(taskRepo, taskDepRepo, timeEventRepo)
	completedCount, err := completeJobTasks(db, taskService, bistroJob.ID, user.ID, space.ID)
	if err != nil {
		return fmt.Errorf("completing job tasks: %w", err)
	}
	log.Printf("  Completed %d tasks in the job with backdated timestamps", completedCount)

	// 8. Create "Console Table" recipe (draft).
	err = createConsoleTableRecipe(recipeService, space.ID, user.ID, stationMap)
	if err != nil {
		return fmt.Errorf("creating Console Table recipe: %w", err)
	}
	log.Println("  Created recipe: Console Table (draft)")

	// 9. Create "Dining Chair" product with sellable variants.
	if err := seedProducts(db, recipeService, space.ID, user.ID); err != nil {
		return fmt.Errorf("seeding products: %w", err)
	}
	log.Println("  Created product: Dining Chair (3 variants)")

	// 10. Create a backlog job — confirmed (quote accepted) but not yet scheduled.
	backlogStatus := models.TaskStatusBacklog
	backlogJob, err := taskService.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:       "Walnut Bookshelf - Henderson Order",
		Description: demoStrPtr("Quote accepted; waiting on shop capacity before scheduling."),
		Type:        models.TaskTypeJob,
		Status:      &backlogStatus,
	})
	if err != nil {
		return fmt.Errorf("creating backlog job: %w", err)
	}
	log.Printf("  Created backlog job: %s (%s)", backlogJob.Title, backlogJob.ID)

	// 11. Stock the material catalog.
	if err := seedMaterials(db, space.ID); err != nil {
		return fmt.Errorf("seeding materials: %w", err)
	}
	log.Println("  Created material catalog (6 materials)")

	return nil
}

// seedMaterials stocks the demo space's material catalog with realistic
// woodshop materials and unit costs.
func seedMaterials(db *gorm.DB, spaceID uuid.UUID) error {
	materialService := services.NewMaterialService(repositories.NewMaterialRepository(db))

	materials := []dtos.CreateMaterialRequest{
		{Name: "8/4 Walnut", Category: demoStrPtr("lumber"), Unit: "board_feet",
			Supplier: demoStrPtr("Goby Walnut"), UnitCost: demoDecPtr("14.50")},
		{Name: "4/4 Cherry", Category: demoStrPtr("lumber"), Unit: "board_feet",
			Supplier: demoStrPtr("Crosscut Hardwoods"), UnitCost: demoDecPtr("8.25")},
		{Name: "4/4 White Oak", Category: demoStrPtr("lumber"), Unit: "board_feet",
			Supplier: demoStrPtr("Crosscut Hardwoods"), UnitCost: demoDecPtr("9.75")},
		{Name: "Titebond III", Category: demoStrPtr("consumable"), Unit: "oz",
			UnitCost: demoDecPtr("0.55")},
		{Name: "Pre-Cat Lacquer", Category: demoStrPtr("finish"), Unit: "gallons",
			Supplier: demoStrPtr("Sherwin-Williams"), UnitCost: demoDecPtr("68.00")},
		{Name: "Brass Screws #8 x 1-1/4", Category: demoStrPtr("hardware"), Unit: "each",
			SKU: demoStrPtr("BRS-8-114"), UnitCost: demoDecPtr("0.32")},
	}
	for i := range materials {
		if _, err := materialService.Create(spaceID, &materials[i]); err != nil {
			return fmt.Errorf("creating material %q: %w", materials[i].Name, err)
		}
	}
	return nil
}

// demoDecPtr parses a decimal literal for seed data.
func demoDecPtr(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

// seedProducts creates the "Dining Chair" product with three sellable
// variants (wood species / finish combinations), each bound to a Dining
// Chair recipe with variable bindings and a customer-facing price.
func seedProducts(
	db *gorm.DB,
	recipeService *services.RecipeService,
	spaceID uuid.UUID,
	userID uuid.UUID,
) error {
	// Recipe the variants point at. Created as a draft; variants bind
	// wood_species/finish variables against it.
	recipeDesc := "Classic hardwood dining chair with mortise & tenon joinery. " +
		"Wood species and finish vary per product variant."
	recipe, _, err := recipeService.CreateRecipeWithTaskTree(
		spaceID, userID, "Dining Chair", &recipeDesc, nil,
	)
	if err != nil {
		return fmt.Errorf("creating Dining Chair recipe: %w", err)
	}

	productService := services.NewProductService(db, repositories.NewRecipeRepository(db))

	productDesc := "Classic hardwood dining chair. Available in multiple wood species and finishes."
	product, err := productService.Create(spaceID, services.CreateProductInput{
		Name:        "Dining Chair",
		Description: &productDesc,
	})
	if err != nil {
		return fmt.Errorf("creating Dining Chair product: %w", err)
	}

	variants := []struct {
		name   string
		wood   string
		finish string
		price  string
	}{
		{name: "Walnut / Spray PU", wood: "Walnut", finish: "Spray PU", price: "640"},
		{name: "Cherry / Hand Oil", wood: "Cherry", finish: "Hand-Rubbed Oil", price: "580"},
		{name: "Oak / Spray PU", wood: "White Oak", finish: "Spray PU", price: "520"},
	}
	for _, v := range variants {
		price, err := decimal.NewFromString(v.price)
		if err != nil {
			return fmt.Errorf("parsing price for variant %q: %w", v.name, err)
		}
		_, err = productService.CreateVariant(spaceID, product.ID, services.CreateVariantInput{
			Name:     v.name,
			RecipeID: &recipe.ID,
			RecipeVariables: models.JSONB{
				"wood_species": v.wood,
				"finish":       v.finish,
			},
			Price: &price,
		})
		if err != nil {
			return fmt.Errorf("creating variant %q: %w", v.name, err)
		}
	}

	return nil
}

// createBistroTableRecipe creates a Bistro Table recipe with 5 steps, assigns
// stations and estimated times, adds dependencies, and publishes it.
func createBistroTableRecipe(
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

	// Get the draft version to find the root task ID.
	versions, err := listRecipeVersions(db, recipe.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(versions) == 0 {
		return nil, nil, fmt.Errorf("no versions found for recipe")
	}
	draftVersion := versions[0]
	rootTaskID := *draftVersion.RootTaskID

	// Define steps: mill → joinery → glue-up → finish → QC
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
			batchSize:   4,    // batch of 4
		},
		{
			title:       "Cut joinery",
			description: "Cut mortise & tenon joints for leg-to-apron connections. Cut slots for breadboard ends.",
			station:     "Joinery",
			estSecs:     3600, // 60 min
			batchSize:   2,    // batch of 2
		},
		{
			title:       "Glue-up and assembly",
			description: "Glue tabletop panels. After cure, attach legs/aprons and breadboard ends.",
			station:     "Glue-Up",
			estSecs:     1800, // 30 min (active time, not cure)
			batchSize:   4,    // batch of 4
		},
		{
			title:       "Sand and finish",
			description: "Progressive sand to 220 grit. Apply 3 coats of conversion varnish with scuff-sand between coats.",
			station:     "Finish",
			estSecs:     5400, // 90 min
			batchSize:   6,    // batch of 6
		},
		{
			title:       "Final QC and pack",
			description: "Inspect all joints, finish quality, and dimensions. Apply felt pads. Wrap and label for shipping.",
			station:     "QC",
			estSecs:     900, // 15 min
			batchSize:   1,   // per piece
		},
	}

	// Add steps to recipe.
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

	// Add dependencies: each step depends on the previous one.
	for i := 1; i < len(stepTasks); i++ {
		err := recipeService.AddStepDependency(
			recipe.ID,
			stepTasks[i].ID,   // this step
			stepTasks[i-1].ID, // depends on previous
		)
		if err != nil {
			return nil, nil, fmt.Errorf("adding dependency %d→%d: %w", i, i-1, err)
		}
	}

	// Publish the recipe.
	published, err := recipeService.PublishVersion(recipe.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("publishing: %w", err)
	}

	// Re-fetch recipe to get updated CurrentVersionID.
	var updatedRecipe models.Recipe
	if err := db.First(&updatedRecipe, "id = ?", recipe.ID).Error; err != nil {
		return nil, nil, err
	}

	return &updatedRecipe, published, nil
}

// completeJobTasks completes the first portion of tasks in the rolled job
// (mill and joinery steps) with realistic backdated timestamps and time variance.
// Returns the count of completed tasks.
func completeJobTasks(
	db *gorm.DB,
	taskService *services.TaskService,
	jobID string,
	userID uuid.UUID,
	spaceID uuid.UUID,
) (int, error) {
	// Load all child tasks of the job.
	children, err := loadJobChildren(db, jobID)
	if err != nil {
		return 0, err
	}

	// We want to complete mill tasks and some joinery tasks.
	// The job has qty 12, so with batch sizes:
	//   Mill: batch_size=4 → 3 tickets
	//   Joinery: batch_size=2 → 6 tickets
	// Complete all 3 mill tickets and 4 of 6 joinery tickets.
	completed := 0
	baseTime := time.Now().Add(-7 * 24 * time.Hour) // started a week ago

	for _, task := range children {
		if task.ParentID == nil || *task.ParentID != jobID {
			continue // only direct children of job root
		}

		isMill := task.DisplayOrder <= 3                              // first 3 are mill (batch of 4, qty 12 → 3 tickets)
		isJoinery := task.DisplayOrder >= 4 && task.DisplayOrder <= 9 // next 6 are joinery

		if isMill || (isJoinery && task.DisplayOrder <= 7) {
			// Backdate: start the task, then complete it.
			startTime := backdateForDemo(baseTime, completed, 4*time.Hour)
			actualSecs := addTimeVariance(task.EstimatedTimeSecs, 0.15)

			if err := backdateTask(db, &task, userID, spaceID, startTime, actualSecs); err != nil {
				return completed, fmt.Errorf("completing task %s: %w", task.ID, err)
			}
			completed++
		}
	}

	return completed, nil
}

// backdateTask directly updates a task to done status with backdated timestamps
// and creates corresponding time events.
func backdateTask(
	db *gorm.DB,
	task *models.Task,
	userID uuid.UUID,
	spaceID uuid.UUID,
	startTime time.Time,
	actualSecs int,
) error {
	endTime := startTime.Add(time.Duration(actualSecs) * time.Second)

	return db.Transaction(func(tx *gorm.DB) error {
		// Update task to done.
		if err := tx.Model(task).Updates(map[string]interface{}{
			"status":           models.TaskStatusDone,
			"started_at":       startTime,
			"completed_at":     endTime,
			"actual_time_secs": actualSecs,
			"updated_at":       endTime,
		}).Error; err != nil {
			return err
		}

		// Create check_in event.
		checkIn := &models.TimeEvent{
			ID:        uuid.New(),
			SpaceID:   spaceID,
			UserID:    userID,
			TaskID:    &task.ID,
			StationID: task.StationID,
			EventType: models.TimeEventTypeCheckIn,
			Source:    models.TimeEventSourceSystem,
			Timestamp: startTime,
		}
		if err := tx.Create(checkIn).Error; err != nil {
			return err
		}

		// Create check_out event.
		checkOut := &models.TimeEvent{
			ID:        uuid.New(),
			SpaceID:   spaceID,
			UserID:    userID,
			TaskID:    &task.ID,
			StationID: task.StationID,
			EventType: models.TimeEventTypeCheckOut,
			Source:    models.TimeEventSourceSystem,
			Timestamp: endTime,
		}
		return tx.Create(checkOut).Error
	})
}

// BackdateForDemo calculates a realistic start time for a task in a demo
// sequence. Each subsequent task starts after a gap that simulates real work
// days with some randomness.
func backdateForDemo(baseTime time.Time, taskIndex int, avgGap time.Duration) time.Time {
	// Add the gap multiplied by task index, plus some jitter.
	offset := time.Duration(taskIndex) * avgGap
	jitter := time.Duration(rand.Int63n(int64(avgGap / 4)))
	return baseTime.Add(offset + jitter)
}

// addTimeVariance returns an actual time based on an estimated time with
// random variance. The variance parameter is the max percentage deviation
// (e.g., 0.15 for +/- 15%).
func addTimeVariance(estimatedSecs *int, variance float64) int {
	if estimatedSecs == nil || *estimatedSecs == 0 {
		return 600 // default 10 minutes
	}
	est := float64(*estimatedSecs)
	// Random factor between (1 - variance) and (1 + variance)
	factor := 1.0 + (rand.Float64()*2-1)*variance
	return int(est * factor)
}

// createConsoleTableRecipe creates a Console Table recipe in draft state.
func createConsoleTableRecipe(
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

	// Add a few steps to make the draft meaningful.
	type stepDef struct {
		title   string
		station string
		estSecs int
	}
	steps := []stepDef{
		{title: "Mill legs and rails", station: "Mill", estSecs: 1800},
		{title: "Cut mortise & tenon joints", station: "Joinery", estSecs: 2400},
		{title: "Build drawer box", station: "Joinery", estSecs: 1800},
		{title: "Assemble carcase", station: "Glue-Up", estSecs: 1200},
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

	return nil
}

// loadJobChildren loads all tasks that are children of the given job.
func loadJobChildren(db *gorm.DB, jobID string) ([]models.Task, error) {
	var tasks []models.Task
	if err := db.Where("id LIKE ? AND id != ?", jobID+".%", jobID).
		Order("display_order ASC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// listRecipeVersions loads all versions for a recipe ordered by version number desc.
func listRecipeVersions(db *gorm.DB, recipeID uuid.UUID) ([]models.RecipeVersion, error) {
	var versions []models.RecipeVersion
	if err := db.Where("recipe_id = ?", recipeID).
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// demoStrPtr returns a pointer to a string.
func demoStrPtr(s string) *string { return &s }

// demoIntPtr returns a pointer to an int.
func demoIntPtr(i int) *int { return &i }
