import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

// Shared space slug set by beforeEach so each test navigates to the
// space-scoped recipes page (which initialises currentSpace via the layout).
let spaceSlug = '';

test.describe('Recipe Authoring (User Story 1)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('create a new recipe and verify it opens with an empty task tree', async ({
    page,
  }) => {
    // Navigate to the space-scoped recipes page (sets currentSpace via layout)
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await expect(page.getByRole('heading', { name: 'Recipes' })).toBeVisible();

    // Click "New Recipe" to open the create dialog
    await page.getByRole('button', { name: 'New Recipe' }).click();

    // Fill in the recipe name
    await page.locator('#recipe-name').fill('Oak Dining Table');
    await page.locator('#recipe-description').fill('Standard 6-seat dining table build');

    // Submit the form
    await page.getByRole('button', { name: 'Create Recipe' }).click();

    // Should navigate to the recipe detail page
    await page.waitForURL(/\/recipes\/.+/);

    // Verify the recipe title is shown
    await expect(page.getByRole('heading', { name: 'Oak Dining Table' })).toBeVisible();

    // Verify it's in Draft status
    await expect(page.getByText('Draft')).toBeVisible();

    // Verify the task tree section is present with "No steps yet" empty state
    await expect(page.getByText('No steps yet.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Add first step' })).toBeVisible();
  });

  test('add top-level steps to the recipe task tree', async ({ page }) => {
    // Create a recipe via the UI first
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Test Recipe');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    // Add first step via the "Add first step" button (empty state)
    await page.getByRole('button', { name: 'Add first step' }).click();
    await page.locator('#add-child-title').fill('Mill');
    await page.getByRole('button', { name: 'Add Step' }).click();

    // Step should appear in the tree
    await expect(page.getByText('Mill')).toBeVisible();
    // Empty state should be gone
    await expect(page.getByText('No steps yet.')).not.toBeVisible();

    // Add more top-level steps via the root "+" button
    // The root header has a Plus button — use the one in the recipe header area
    const rootAddButton = page
      .locator('div')
      .filter({ hasText: /^Test Recipe/ })
      .getByRole('button')
      .first();
    await rootAddButton.click();
    await page.locator('#add-child-title').fill('Joinery');
    await page.getByRole('button', { name: 'Add Step' }).click();

    await expect(page.getByText('Joinery')).toBeVisible();

    // Add a third step
    await rootAddButton.click();
    await page.locator('#add-child-title').fill('Assembly');
    await page.getByRole('button', { name: 'Add Step' }).click();

    await expect(page.getByText('Assembly')).toBeVisible();

    // Verify all three steps are visible in the tree
    await expect(page.getByText('Mill')).toBeVisible();
    await expect(page.getByText('Joinery')).toBeVisible();
    await expect(page.getByText('Assembly')).toBeVisible();
  });

  test('add nested sub-steps under a parent step', async ({ page }) => {
    // Create recipe and add a top-level step
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Nested Steps Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    // Add top-level "Mill" step
    await page.getByRole('button', { name: 'Add first step' }).click();
    await page.locator('#add-child-title').fill('Mill');
    await page.getByRole('button', { name: 'Add Step' }).click();
    await expect(page.getByText('Mill')).toBeVisible();

    // Hover over "Mill" to reveal action buttons, then click the add-child button
    const millRow = page.locator('[role="button"]').filter({ hasText: 'Mill' });
    await millRow.hover();

    // The first action button in the hover group is the "Add child step" button
    const addChildBtn = millRow.locator('button').filter({ has: page.locator('svg') }).first();
    await addChildBtn.click();

    // Add a sub-step
    await page.locator('#add-child-title').fill('Cut to length');
    await page.getByRole('button', { name: 'Add Step' }).click();

    // Verify sub-step is visible in the tree
    await expect(page.getByText('Cut to length')).toBeVisible();

    // Add another sub-step
    await millRow.hover();
    await addChildBtn.click();
    await page.locator('#add-child-title').fill('Plane faces');
    await page.getByRole('button', { name: 'Add Step' }).click();

    await expect(page.getByText('Plane faces')).toBeVisible();
  });

  test('add dependencies between sibling steps', async ({ page }) => {
    // Create recipe with two top-level steps
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Dependency Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    // Add "Mill" step
    await page.getByRole('button', { name: 'Add first step' }).click();
    await page.locator('#add-child-title').fill('Mill');
    await page.getByRole('button', { name: 'Add Step' }).click();
    await expect(page.getByText('Mill')).toBeVisible();

    // Add "Joinery" step
    const rootAddButton = page
      .locator('div')
      .filter({ hasText: /^Dependency Test/ })
      .getByRole('button')
      .first();
    await rootAddButton.click();
    await page.locator('#add-child-title').fill('Joinery');
    await page.getByRole('button', { name: 'Add Step' }).click();
    await expect(page.getByText('Joinery')).toBeVisible();

    // Open dependency manager for "Joinery" (hover to reveal link icon button)
    const joineryRow = page.locator('[role="button"]').filter({ hasText: 'Joinery' });
    await joineryRow.hover();

    // The link icon button is the second action button in the hover group
    const depBtn = joineryRow.locator('button').nth(1);
    await depBtn.click();

    // The dependency dialog should open
    await expect(page.getByText('Manage Dependencies')).toBeVisible();

    // "Mill" should appear as an available blocker — click "Add" next to it
    const millBlockerRow = page.locator('div').filter({ hasText: /^Mill/ }).last();
    await millBlockerRow.getByRole('button', { name: 'Add' }).click();

    // "Mill" should now appear under "Blocked by"
    await expect(page.getByRole('heading', { name: 'Blocked by' })).toBeVisible();

    // Close the dialog
    await page.getByRole('button', { name: 'Done' }).click();

    // Verify the dependency badge (link icon + "1") is visible on the Joinery row
    await expect(joineryRow.locator('text=1')).toBeVisible();
  });

  test('recipe appears in the recipes list after creation', async ({ page }) => {
    // Create a recipe
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Walnut Bookshelf');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    // Navigate back to recipes list (client-side navigation preserves currentSpace)
    await page.getByRole('button', { name: 'Back to Recipes' }).click();
    await page.waitForURL('/recipes');

    // The recipe should be in the list
    await expect(page.getByText('Walnut Bookshelf')).toBeVisible();
    await expect(page.getByText('Draft')).toBeVisible();
  });
});
