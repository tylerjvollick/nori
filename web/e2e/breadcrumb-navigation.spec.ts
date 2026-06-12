import { test, expect } from '@playwright/test';
import { resetSpace } from './helpers/reset';

let spaceSlug = '';

test.describe('Breadcrumb navigation on detail views', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('recipe detail shows breadcrumb segments and no tab bar', async ({ page }) => {
    // Create a recipe
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Breadcrumb Test Recipe');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Assert breadcrumb segments
    const breadcrumbNav = page.locator('nav[aria-label="breadcrumb"]');
    await expect(breadcrumbNav).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-link"]', { hasText: 'Spaces' })).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-link"]', { hasText: 'Recipes' })).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-page"]', { hasText: 'Breadcrumb Test Recipe' })).toBeVisible();

    // Tab bar should NOT be visible
    await expect(page.locator('[role="tablist"]')).not.toBeVisible();

    // "Back to Recipes" link should NOT exist
    await expect(page.getByText('Back to Recipes')).not.toBeVisible();
  });

  test('job detail shows breadcrumb segments and no space-level tab bar', async ({ page }) => {
    // Create a job
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Breadcrumb Test Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Assert breadcrumb segments
    const breadcrumbNav = page.locator('nav[aria-label="breadcrumb"]');
    await expect(breadcrumbNav).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-link"]', { hasText: 'Spaces' })).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-page"]', { hasText: 'Breadcrumb Test Job' })).toBeVisible();

    // Space-level tabs (Jobs/Recipes/etc) should NOT be visible
    await expect(page.getByRole('tab', { name: 'Recipes' })).not.toBeVisible();
    // Job detail should show Tasks/Cost tabs
    await expect(page.getByRole('tab', { name: 'Tasks' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Cost' })).toBeVisible();
  });

  test('task under job shows 4-segment breadcrumb with job and task names', async ({ page }) => {
    // Create a job
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Parent Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Wait for graph to load and add a child task
    const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });
    await addNodeBtn.click();

    // Wait for the new node's inline input, name it, commit
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 10000 });
    await inlineInput.fill('Child Task');
    await page.keyboard.press('Enter');

    // Wait for node to render with committed title
    const childNode = page.locator('.svelte-flow__node', { hasText: 'Child Task' });
    await expect(childNode).toBeVisible({ timeout: 10000 });

    // Get the child task ID from the graph node's data attribute
    const childTaskId = await childNode.getAttribute('data-id');
    expect(childTaskId).toBeTruthy();

    // Navigate directly to the child task
    await page.goto(`/spaces/${spaceSlug}/${childTaskId}`);
    await page.waitForLoadState('networkidle');

    // Assert 4-segment breadcrumb: Spaces > SpaceName > Parent Job > Child Task
    const breadcrumbNav = page.locator('nav[aria-label="breadcrumb"]');
    await expect(breadcrumbNav).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-link"]', { hasText: 'Spaces' })).toBeVisible();
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-link"]', { hasText: 'Parent Job' })).toBeVisible({ timeout: 10000 });
    await expect(breadcrumbNav.locator('[data-slot="breadcrumb-page"]', { hasText: 'Child Task' })).toBeVisible();

    // Tab bar should NOT be visible
    await expect(page.locator('[role="tablist"]')).not.toBeVisible();
  });

  test('clicking parent-job breadcrumb from a child task re-fetches the parent (nori-zv0.7)', async ({
    page,
  }) => {
    // Create a job
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Refetch Parent Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Add a child task to the job
    const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });
    await addNodeBtn.click();
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 10000 });
    await inlineInput.fill('Refetch Child Task');
    await page.keyboard.press('Enter');
    const childNode = page.locator('.svelte-flow__node', { hasText: 'Refetch Child Task' });
    await expect(childNode).toBeVisible({ timeout: 10000 });
    const childTaskId = await childNode.getAttribute('data-id');
    expect(childTaskId).toBeTruthy();

    // Open the child task detail page directly
    await page.goto(`/spaces/${spaceSlug}/${childTaskId}`);
    await page.waitForLoadState('networkidle');

    const breadcrumbNav = page.locator('nav[aria-label="breadcrumb"]');
    await expect(
      breadcrumbNav.locator('[data-slot="breadcrumb-page"]', { hasText: 'Refetch Child Task' }),
    ).toBeVisible({ timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 2, name: 'Refetch Child Task' }),
    ).toBeVisible({ timeout: 10000 });

    // Click the parent-job breadcrumb link — this is in-app navigation that reuses
    // the [taskId] component, which is exactly where the stale-data bug lived.
    const parentLink = breadcrumbNav.locator('[data-slot="breadcrumb-link"]', {
      hasText: 'Refetch Parent Job',
    });
    const parentHref = await parentLink.getAttribute('href');
    expect(parentHref).toMatch(new RegExp(`^/spaces/${spaceSlug}/[^/]+$`));
    await parentLink.click();

    await page.waitForURL((url) => url.pathname === parentHref, { timeout: 10000 });

    // The page must now render the PARENT job, not the stale child.
    await expect(
      breadcrumbNav.locator('[data-slot="breadcrumb-page"]', { hasText: 'Refetch Parent Job' }),
    ).toBeVisible({ timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 2, name: 'Refetch Parent Job' }),
    ).toBeVisible({ timeout: 10000 });
    // The parent's tree re-fetched: the child task node is shown in the graph.
    await expect(
      page.locator('.svelte-flow__node', { hasText: 'Refetch Child Task' }),
    ).toBeVisible({ timeout: 10000 });
    // The child task is no longer the current page.
    await expect(
      breadcrumbNav.locator('[data-slot="breadcrumb-page"]', { hasText: 'Refetch Child Task' }),
    ).toHaveCount(0);
  });

  test('clicking SpaceName breadcrumb navigates to space view with tab bar', async ({ page }) => {
    // Create a recipe to get to a detail view
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Nav Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Tab bar should NOT be visible on detail view
    await expect(page.locator('[role="tablist"]')).not.toBeVisible();

    // Click the SpaceName breadcrumb segment
    const breadcrumbNav = page.locator('nav[aria-label="breadcrumb"]');
    const spaceLink = breadcrumbNav.locator('[data-slot="breadcrumb-link"]').filter({ hasNotText: 'Spaces' }).filter({ hasNotText: 'Recipes' }).first();
    await spaceLink.click();

    // Should navigate to the space view
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/`));

    // Tab bar should be visible on space-level view
    await expect(page.locator('[role="tablist"]')).toBeVisible({ timeout: 5000 });
  });
});
