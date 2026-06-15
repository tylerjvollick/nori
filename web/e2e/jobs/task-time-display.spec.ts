import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from '../helpers/reset';
import { env } from '../helpers/env';

let spaceSlug = '';
let spaceId = '';

/** Read the logged-in test user's auth token from localStorage. */
async function authToken(page: Page): Promise<string> {
  await page.goto(env.baseURL);
  return page.evaluate(() => localStorage.getItem('accessToken') ?? '');
}

/** Create a job, then add the named child tasks via the graph. */
async function createJobWithChildren(
  page: Page,
  jobTitle: string,
  childTitles: string[],
): Promise<{ jobId: string; childIds: string[] }> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(jobTitle);
  await page.getByRole('button', { name: 'Create Job' }).click();
  // Wait for the job's own detail URL (task-…), not the intermediate /jobs route.
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/task-[0-9a-f]+`));

  await page.getByRole('tab', { name: 'Graph' }).click();
  const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
  await expect(addNodeBtn).toBeVisible({ timeout: 10_000 });

  const jobId = page.url().split('?')[0].split('/').pop()!;
  expect(jobId).toMatch(/^task-[0-9a-f]+$/);

  const childIds: string[] = [];
  for (const title of childTitles) {
    await addNodeBtn.click();
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 10_000 });
    await inlineInput.fill(title);
    await page.keyboard.press('Enter');

    const childNode = page.locator('.svelte-flow__node', { hasText: title });
    await expect(childNode).toBeVisible({ timeout: 10_000 });
    const childId = await childNode.getAttribute('data-id');
    expect(childId).toBeTruthy();
    childIds.push(childId!);
  }

  return { jobId, childIds };
}

/** Create a job with one child task; return the child's task id. */
async function createJobWithChild(
  page: Page,
  jobTitle: string,
  childTitle: string,
): Promise<string> {
  const { childIds } = await createJobWithChildren(page, jobTitle, [childTitle]);
  return childIds[0];
}

/** Seed a finished (stopped) time entry of `elapsedSecs` on a task via the API. */
async function addTimeEntry(token: string, taskId: string, elapsedSecs: number): Promise<void> {
  const res = await fetch(`${env.apiURL}/api/v1/spaces/${spaceId}/tasks/${taskId}/time-entries`, {
    method: 'POST',
    headers: { Cookie: `nori_token=${token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      startedAt: new Date(Date.now() - elapsedSecs * 1000).toISOString(),
      endedAt: new Date().toISOString(),
      elapsedSecs,
    }),
  });
  if (!res.ok) {
    throw new Error(`create time entry failed (${res.status}): ${await res.text()}`);
  }
}

// nori-e5q: the timer slot showed the lifetime sum even when stopped, and
// "Total Recorded" read the stale task.actualTimeSeconds field. These must swap:
// the timer shows the current session only; Total Recorded shows the entry sum.
test.describe('Task detail: time display', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
    spaceId = result.spaceId;
  });

  test('stopped task shows an empty timer and Total Recorded equals the entry sum', async ({
    page,
  }) => {
    const childId = await createJobWithChild(page, 'Time Parent Job', 'Tracked Task');
    const token = await authToken(page);

    // Three stopped entries totalling 3h (1h each) — no running timer.
    await addTimeEntry(token, childId, 3600);
    await addTimeEntry(token, childId, 3600);
    await addTimeEntry(token, childId, 3600);

    await page.goto(`/spaces/${spaceSlug}/${childId}`);
    await expect(page.getByRole('heading', { level: 2, name: 'Tracked Task' })).toBeVisible({
      timeout: 10_000,
    });

    // Timer slot is empty — there is no active session.
    await expect(page.getByTestId('timer-session')).toHaveCount(0);
    // A Start button is still offered.
    await expect(page.getByTestId('timer-start')).toBeVisible();

    // Total Recorded reflects the sum of all entries (3h), not actualTimeSeconds.
    await expect(page.getByTestId('logged-time')).toHaveText('3h 0m', { timeout: 10_000 });
  });

  test('parent job Total Recorded rolls up time from child tasks', async ({ page }) => {
    // Heavier flow (job + two graph children + a heavy Overview page).
    test.setTimeout(60_000);

    const { jobId, childIds } = await createJobWithChildren(page, 'Rollup Job', [
      'Cut Parts',
      'Assemble',
    ]);
    const token = await authToken(page);

    // 2h on the first child, 1h on the second; nothing logged on the job itself.
    await addTimeEntry(token, childIds[0], 2 * 3600);
    await addTimeEntry(token, childIds[1], 1 * 3600);

    // Open the parent job's detail page (Overview tab shows the job root panel).
    await page.goto(`/spaces/${spaceSlug}/${jobId}`);
    await expect(page.getByRole('heading', { level: 2, name: 'Rollup Job' })).toBeVisible({
      timeout: 15_000,
    });

    // Total Recorded shows the rolled-up total across descendants (2h + 1h = 3h),
    // even though the job has no direct time entries of its own. The Overview tab
    // renders the panel twice (desktop + a hidden mobile drawer) — assert the
    // visible one.
    await expect(page.locator('[data-testid="logged-time"]:visible')).toHaveText('3h 0m', {
      timeout: 15_000,
    });
  });

  test('starting a timer shows the live session in the timer slot', async ({ page }) => {
    const childId = await createJobWithChild(page, 'Live Timer Job', 'Running Task');
    const token = await authToken(page);

    // One prior stopped entry of 1h.
    await addTimeEntry(token, childId, 3600);

    await page.goto(`/spaces/${spaceSlug}/${childId}`);
    await expect(page.getByRole('heading', { level: 2, name: 'Running Task' })).toBeVisible({
      timeout: 10_000,
    });

    // Start a timer — the session display appears (begins near 0, not the 1h total).
    await page.getByTestId('timer-start').click();
    const session = page.getByTestId('timer-session');
    await expect(session).toBeVisible({ timeout: 10_000 });
    await expect(session).not.toContainText('1h');

    // Total Recorded still reflects the lifetime sum including the prior hour.
    await expect(page.getByTestId('logged-time')).toContainText('1h', { timeout: 10_000 });
  });
});
