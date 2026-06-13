import { test, expect, type Page, type Locator } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

// A minimal valid 1x1 PNG, base64-encoded.
const PNG_1x1 = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC',
  'base64',
);

function imageFile(name = 'test-photo.png') {
  return { name, mimeType: 'image/png', buffer: PNG_1x1 };
}

/**
 * The task detail panel — scoped by the Sub-tasks heading it contains (same
 * approach as sub-task-editing.spec.ts). Both Photos and Sub-tasks render here.
 */
function detailPanel(page: Page): Locator {
  return page
    .locator('div.border-r')
    .filter({ has: page.getByRole('heading', { name: 'Sub-tasks' }) })
    .first();
}

/**
 * Create a job, add a child task, and open that child's own detail page so its
 * detail panel (with Photos + Sub-tasks sections) renders as the left pane.
 */
async function createJobAndSelectChildTask(page: Page, jobTitle: string): Promise<Locator> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(jobTitle);
  await page.getByRole('button', { name: 'Create Job' }).click();
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

  await page.getByRole('tab', { name: 'Graph' }).click();
  const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
  await expect(addNodeBtn).toBeVisible({ timeout: 10000 });
  await addNodeBtn.click();
  const input = page.locator('.svelte-flow__node input');
  await expect(input).toBeVisible({ timeout: 10000 });
  await page.keyboard.press('Enter');
  await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10000 });

  const childId = await page.locator('.svelte-flow__node').first().getAttribute('data-id');
  await page.goto(`/spaces/${spaceSlug}/${childId}`);
  await page.waitForLoadState('networkidle');

  const panel = detailPanel(page);
  await expect(panel.getByRole('heading', { name: 'Sub-tasks' })).toBeVisible({ timeout: 10000 });
  return panel;
}

async function addSubTask(panel: Locator, title: string): Promise<void> {
  await panel.getByText('Add sub-task').click();
  await panel.getByPlaceholder('Sub-task title').fill(title);
  await panel.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(panel.getByText(title)).toBeVisible();
}

test.describe('Photo attachment on tasks and sub-steps', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('upload a photo on a task, see it, then delete it', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'Photo Task A');

    const photos = panel.getByTestId('task-photos');
    await expect(photos.getByText('No photos yet.')).toBeVisible();

    // Set the file directly on the hidden input (no target dependency).
    await photos.getByTestId('task-photos-input').setInputFiles(imageFile());

    // The thumbnail appears in the grid.
    const grid = photos.getByTestId('task-photos-grid');
    await expect(grid).toBeVisible({ timeout: 10000 });
    const img = grid.locator('img');
    await expect(img).toHaveCount(1, { timeout: 10000 });
    await expect(img.first()).toHaveAttribute('src', /\/uploads\/task-media\//);
    await expect(photos.getByText('No photos yet.')).not.toBeVisible();

    // Delete it.
    const photoCell = grid.locator('div.group\\/photo').first();
    await photoCell.hover();
    await photoCell.getByTestId('task-photo-delete').click();

    await expect(photos.getByText('No photos yet.')).toBeVisible({ timeout: 10000 });
  });

  test('uploaded photo survives a page reload', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'Photo Task B');
    const photos = panel.getByTestId('task-photos');

    await photos.getByTestId('task-photos-input').setInputFiles(imageFile());
    await expect(photos.getByTestId('task-photos-grid').locator('img')).toHaveCount(1, {
      timeout: 10000,
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    const reloaded = detailPanel(page).getByTestId('task-photos');
    await expect(reloaded.getByTestId('task-photos-grid').locator('img')).toHaveCount(1, {
      timeout: 10000,
    });
  });

  test('attach an image to a sub-step', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'Photo Task C');
    await addSubTask(panel, 'Glue the joints');

    const row = panel.locator('.group\\/item', { hasText: 'Glue the joints' });
    await row.hover();

    // Clicking "Attach image" opens the file chooser; provide the file to it.
    const [chooser] = await Promise.all([
      page.waitForEvent('filechooser'),
      row.getByTestId('subtask-image-add').click(),
    ]);
    await chooser.setFiles(imageFile('step.png'));

    // The row's thumbnail now shows the uploaded image.
    const thumb = row.locator('img');
    await expect(thumb).toBeVisible({ timeout: 10000 });
    await expect(thumb).toHaveAttribute('src', /\/uploads\/sub-task-images\//);
  });
});
