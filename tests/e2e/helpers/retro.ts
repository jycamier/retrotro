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
  await page.getByRole('button', { name: /Nouvelle session/i }).first().click();
  // Session type defaults to "retro" — just fill in the form
  const retroName = `E2E Retro ${Date.now()}`;
  await page.locator('input[type="text"]').first().clear();
  await page.locator('input[type="text"]').first().fill(retroName);
  // Select first available template
  const templateSelect = page.locator('select');
  await templateSelect.selectOption({ index: 1 });
  await page.getByRole('button', { name: /^Créer$/i }).click();
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
 * Create a Lean Coffee session from the Dev Team page.
 * The LC is auto-started and navigates directly to the board.
 * Returns the lean coffee board URL.
 */
export async function createLeanCoffee(page: Page): Promise<string> {
  // Navigate to the existing Dev Team
  await page.getByText('Dev Team').click();
  await page.waitForURL(/\/teams\//);

  // Open creation modal
  await page.getByRole('button', { name: /Nouvelle session/i }).first().click();

  // Select Lean Coffee type (use the button, not the template option in the select)
  await page.getByRole('button', { name: /Lean Coffee/i }).click();

  // Fill name
  const lcName = `E2E LC ${Date.now()}`;
  await page.locator('input[type="text"]').first().clear();
  await page.locator('input[type="text"]').first().fill(lcName);

  // Set timebox to 2 min for faster tests
  const timeboxInput = page.locator('input[type="number"]');
  await timeboxInput.clear();
  await timeboxInput.fill('2');

  // Create — LC auto-starts and navigates
  await page.getByRole('button', { name: /^Créer$/i }).click();
  await page.waitForURL(/\/leancoffee\//, { timeout: 15_000 });
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
    // Normalize whitespace to single spaces for easier matching
    const normalized = text?.replace(/\s+/g, ' ') || '';
    // Matches "3 / 6" (waiting room), "3/6", "(3)" (board header)
    const hasCount = normalized.includes(`${count} /`) || normalized.includes(`${count}/`) || normalized.includes(`(${count})`);
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
