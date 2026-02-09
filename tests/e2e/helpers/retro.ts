import { type Page, expect } from '@playwright/test';

/**
 * Use the existing Dev Team and create a retrospective, then start it.
 * Returns the retro board URL.
 */
export async function createTeamAndRetro(page: Page): Promise<string> {
  // Navigate to the existing Dev Team
  await page.getByText('Dev Team').click();
  await page.waitForURL(/\/teams\//);

  // Create retro
  await page.getByRole('button', { name: /New Retrospective/i }).click();
  const retroName = `E2E Retro ${Date.now()}`;
  await page.getByPlaceholder('Sprint 42 Retrospective').fill(retroName);
  // Select first available template
  const templateSelect = page.locator('select');
  await templateSelect.selectOption({ index: 1 });
  await page.getByRole('button', { name: /^Create$/i }).click();
  await page.waitForTimeout(1_000);

  // Navigate to the retro detail page
  await page.getByText(retroName).click();
  await page.waitForURL(/\/retros\//);

  // Start the retro — this navigates directly to the retro board
  await page.getByRole('button', { name: /Démarrer la Retro/i }).click();
  await page.waitForURL(/\/retro\//, { timeout: 10_000 });
  await page.waitForTimeout(2_000);

  return page.url();
}

/**
 * Join an existing retro by navigating to its URL.
 */
export async function joinRetro(page: Page, retroUrl: string): Promise<void> {
  await page.goto(retroUrl);
  await page.waitForTimeout(3_000);
}

/**
 * Wait until the expected number of participants is displayed.
 */
export async function waitForParticipantCount(page: Page, count: number): Promise<void> {
  await expect(async () => {
    const text = await page.textContent('body');
    // Matches both "3/6 participants connectés" (waiting room) and "Participants (3)" (board)
    const hasCount = text?.includes(`${count}/`) || text?.includes(`(${count})`);
    expect(hasCount).toBeTruthy();
  }).toPass({ timeout: 15_000 });
}

/**
 * Advance to the next phase (facilitator only).
 */
export async function nextPhase(page: Page): Promise<void> {
  const btn = page.getByRole('button', { name: /Continuer vers|Phase suivante|Démarrer la rétrospective/i });
  await btn.waitFor({ timeout: 10_000 });
  await btn.click();
}
