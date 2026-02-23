import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro, waitForParticipantCount, nextPhase } from './helpers/retro';

let ctx1: { context: BrowserContext; page: Page };
let ctx2: { context: BrowserContext; page: Page };
let ctx3: { context: BrowserContext; page: Page };
let retroUrl: string;

test.describe('Multi-user retrospective', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    ctx1 = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctx2 = await createAuthenticatedContext(browser, DEV_USERS.user1);
    ctx3 = await createAuthenticatedContext(browser, DEV_USERS.user2);
  });

  test.afterAll(async () => {
    await ctx1?.context?.close();
    await ctx2?.context?.close();
    await ctx3?.context?.close();
  });

  test('Scenario 1: Join a retro with multiple users', async () => {
    // User1 (admin) creates retro in Dev Team and starts it
    retroUrl = await createTeamAndRetro(ctx1.page);

    // User2 and User3 join
    await joinRetro(ctx2.page, retroUrl);
    await joinRetro(ctx3.page, retroUrl);

    // All 3 should see the participant count
    await waitForParticipantCount(ctx1.page, 3);
    await waitForParticipantCount(ctx2.page, 3);
    await waitForParticipantCount(ctx3.page, 3);
  });

  test('Scenario 2: Item creation and broadcast', async () => {
    // We're in the waiting phase. Facilitator starts the retro.
    await ctx1.page.getByRole('button', { name: /Démarrer la rétrospective/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Now in icebreaker phase — skip to brainstorm
    await ctx1.page.getByRole('button', { name: /Continuer vers Brainstorm/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Wait for brainstorm phase
    await expect(ctx1.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 10_000 });

    // User1 creates an item
    const itemInput = ctx1.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await itemInput.fill('E2E test item from User1');
    await itemInput.press('Enter');

    // User1 sees their own item
    await expect(ctx1.page.getByText('E2E test item from User1')).toBeVisible({ timeout: 5_000 });

    // User2 creates an item
    const item2Input = ctx2.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await item2Input.fill('E2E test item from User2');
    await item2Input.press('Enter');

    await expect(ctx2.page.getByText('E2E test item from User2')).toBeVisible({ timeout: 5_000 });
  });

  test('Scenario 3: Voting', async () => {
    // Advance brainstorm → group
    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(2_000);

    // Advance group → vote
    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(2_000);

    await expect(ctx1.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });

    // Items are now revealed - find first vote button
    const voteButtons1 = ctx1.page.locator('button:has(svg.lucide-thumbs-up)');
    const firstVoteBtn = voteButtons1.first();
    await firstVoteBtn.waitFor({ timeout: 5_000 });
    await firstVoteBtn.click();

    await ctx2.page.waitForTimeout(2_000);

    // User2 also votes
    const voteButtons2 = ctx2.page.locator('button:has(svg.lucide-thumbs-up)');
    await voteButtons2.first().click();

    await ctx3.page.waitForTimeout(2_000);
  });

  test('Scenario 4: Icebreaker / Mood', async () => {
    // Icebreaker was skipped in the main flow.
    // A standalone icebreaker test would require a separate retro.
    test.skip();
  });

  test('Scenario 5: Participant disconnection', async () => {
    // User3 closes their page
    await ctx3.page.close();

    // Wait for grace period (10s backend) + cross-pod relay + broadcast propagation
    // With 2 replicas, the leave event must propagate via PGBridge LISTEN/NOTIFY
    await expect(async () => {
      const text1 = await ctx1.page.textContent('body');
      const has2 = text1?.includes('(2)') || text1?.includes('2/');
      expect(has2).toBeTruthy();
    }).toPass({ timeout: 30_000 });

    await expect(async () => {
      const text2 = await ctx2.page.textContent('body');
      const has2 = text2?.includes('(2)') || text2?.includes('2/');
      expect(has2).toBeTruthy();
    }).toPass({ timeout: 15_000 });

    // Reopen user3 for cleanup
    ctx3.page = await ctx3.context.newPage();
  });

  test('Scenario 6: Timer synchronization', async () => {
    // Advance vote → discuss
    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(2_000);

    await expect(ctx1.page.getByText('Discuss', { exact: true })).toBeVisible({ timeout: 10_000 });

    // Facilitator starts timer
    const timerBtn = ctx1.page.getByRole('button', { name: /timer|minuteur/i });
    if (await timerBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await timerBtn.click();
      await expect(ctx2.page.locator('text=/\\d+:\\d+/')).toBeVisible({ timeout: 5_000 });
    }
  });
});
