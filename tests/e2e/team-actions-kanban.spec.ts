import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro, nextPhase } from './helpers/retro';

let ctxAdmin: { context: BrowserContext; page: Page };
let ctxUser1: { context: BrowserContext; page: Page };
let retroUrl: string;

test.describe('Team Actions Kanban', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    ctxAdmin = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctxUser1 = await createAuthenticatedContext(browser, DEV_USERS.user1);
  });

  test.afterAll(async () => {
    await ctxAdmin?.context?.close();
    await ctxUser1?.context?.close();
  });

  test('Setup: Create retro with actions and complete it', async () => {
    // Admin creates retro
    retroUrl = await createTeamAndRetro(ctxAdmin.page);

    // User1 joins
    await joinRetro(ctxUser1.page, retroUrl);

    // Start retro (waiting → icebreaker)
    await ctxAdmin.page.getByRole('button', { name: /Démarrer la rétrospective/i }).click();
    await ctxAdmin.page.waitForTimeout(1_000);

    // Skip icebreaker → brainstorm
    await ctxAdmin.page.getByRole('button', { name: /Continuer vers Brainstorm/i }).click();
    await ctxAdmin.page.waitForTimeout(1_000);

    // Wait for brainstorm phase
    await expect(ctxAdmin.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 10_000 });

    // Create items in Went Well column
    const itemInput = ctxAdmin.page.locator('input[placeholder="Ajouter un élément..."]').first();
    
    await itemInput.fill('Fix bug in login');
    await itemInput.press('Enter');
    await ctxAdmin.page.waitForTimeout(500);

    await itemInput.fill('Update documentation');
    await itemInput.press('Enter');
    await ctxAdmin.page.waitForTimeout(500);

    await itemInput.fill('Add new feature');
    await itemInput.press('Enter');
    await ctxAdmin.page.waitForTimeout(1_000);

    // Advance to action phase
    await nextPhase(ctxAdmin.page); // brainstorm → group
    await ctxAdmin.page.waitForTimeout(1_000);
    
    await nextPhase(ctxAdmin.page); // group → vote
    await ctxAdmin.page.waitForTimeout(1_000);
    
    await nextPhase(ctxAdmin.page); // vote → discuss
    await ctxAdmin.page.waitForTimeout(1_000);
    
    await nextPhase(ctxAdmin.page); // discuss → action
    await ctxAdmin.page.waitForTimeout(2_000);

    await expect(ctxAdmin.page.getByText(/action/i)).toBeVisible({ timeout: 10_000 });

    // Create actions
    await ctxAdmin.page.fill('input[placeholder*="action"]', 'Complete the login fix');
    await ctxAdmin.page.press('input[placeholder*="action"]', 'Enter');
    await ctxAdmin.page.waitForTimeout(500);

    await ctxAdmin.page.fill('input[placeholder*="action"]', 'Write API docs');
    await ctxAdmin.page.press('input[placeholder*="action"]', 'Enter');
    await ctxAdmin.page.waitForTimeout(500);

    await ctxAdmin.page.fill('input[placeholder*="action"]', 'Deploy to production');
    await ctxAdmin.page.press('input[placeholder*="action"]', 'Enter');
    await ctxAdmin.page.waitForTimeout(1_000);

    // End the retro
    await nextPhase(ctxAdmin.page); // action → end
    await ctxAdmin.page.waitForTimeout(2_000);
  });

  test('Team actions page shows actions from completed retros', async () => {
    // Get team ID from URL
    const retroId = retroUrl.split('/').pop();
    
    // Navigate to team page (we need to get the team ID first)
    // Since we created a retro, let's navigate to a known team slug
    await ctxAdmin.page.goto('/teams/dev-team');
    await ctxAdmin.page.waitForTimeout(2_000);

    // Click on Actions button
    await ctxAdmin.page.getByRole('button', { name: /Actions/i }).click();
    await ctxAdmin.page.waitForTimeout(2_000);

    // Should see the Kanban board
    await expect(ctxAdmin.page.getByText('À faire')).toBeVisible();
    await expect(ctxAdmin.page.getByText('En cours')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Terminé')).toBeVisible();

    // Should see the 3 actions we created
    await expect(ctxAdmin.page.getByText('Complete the login fix')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Write API docs')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Deploy to production')).toBeVisible();
  });

  test('Filter actions by retrospective', async () => {
    // Already on actions page
    await expect(ctxAdmin.page.getByText('À faire')).toBeVisible();

    // Check filter dropdown exists
    const filterDropdown = ctxAdmin.page.locator('select');
    await expect(filterDropdown).toBeVisible();

    // Should have "Tous les rétro" option
    await expect(filterDropdown).toContainText('Tous les rétro');
  });

  test('Action card shows retro name', async () => {
    // Check that action cards show which retro they came from
    await expect(ctxAdmin.page.getByText(/Sprint/).first()).toBeVisible({ timeout: 5_000 });
  });

  test('Can toggle action completion', async () => {
    // Find an action and click the toggle button
    const actionCard = ctxAdmin.page.getByText('Complete the login fix').locator('..');
    
    // Click the "Marquer terminé" button
    await actionCard.getByText('Marquer terminé').click();
    await ctxAdmin.page.waitForTimeout(500);

    // Should now show "Terminé"
    await expect(actionCard.getByText('Terminé')).toBeVisible();
  });
});
