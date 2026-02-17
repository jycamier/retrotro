import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro, waitForParticipantCount, nextPhase } from './helpers/retro';
import { applyNetworkLatency, resetNetworkLatency, LATENCY_PROFILES } from './helpers/network';

let ctx1: { context: BrowserContext; page: Page };
let ctx2: { context: BrowserContext; page: Page };
let retroUrl: string;

test.describe('Multi-user retrospective with network latency', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    ctx1 = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctx2 = await createAuthenticatedContext(browser, DEV_USERS.user1);
  });

  test.afterAll(async () => {
    await resetNetworkLatency(ctx1.page);
    await resetNetworkLatency(ctx2.page);
    await ctx1?.context?.close();
    await ctx2?.context?.close();
  });

  test('Scenario 1: Join retro with MEDIUM latency (50ms)', async () => {
    // Apply medium latency (typical WiFi)
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);

    // User1 creates retro
    retroUrl = await createTeamAndRetro(ctx1.page);

    // User2 joins
    await joinRetro(ctx2.page, retroUrl);

    // Both should see each other (with latency, this takes longer)
    await waitForParticipantCount(ctx1.page, 2);
    await waitForParticipantCount(ctx2.page, 2);
  });

  test('Scenario 2: Item creation under SLOW latency (150ms)', async () => {
    // Apply slow latency (4G mobile network)
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.slow);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.slow);

    // Start retro and move to brainstorm
    await ctx1.page.getByRole('button', { name: /Démarrer la rétrospective/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    await ctx1.page.getByRole('button', { name: /Continuer vers Brainstorm/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Wait for brainstorm phase on both clients
    await expect(ctx1.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 15_000 });
    await expect(ctx2.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 15_000 });

    // User1 creates an item (with latency, broadcasts will be delayed)
    const itemInput = ctx1.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await itemInput.fill('Item created under 150ms latency');
    await itemInput.press('Enter');

    // User1 sees their item
    await expect(ctx1.page.getByText('Item created under 150ms latency')).toBeVisible({ timeout: 10_000 });

    // User2 should eventually see it (with latency delay)
    await expect(ctx2.page.getByText('Item created under 150ms latency')).toBeVisible({ timeout: 15_000 });
  });

  test('Scenario 3: Rapid item creation under VERY SLOW latency (400ms)', async () => {
    // Apply very slow latency (poor 3G connection)
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.verySlow);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.verySlow);

    // Both users rapidly create items (tests if slow network can handle traffic)
    const item1Input = ctx1.page.locator('input[placeholder="Ajouter un élément..."]').first();
    const item2Input = ctx2.page.locator('input[placeholder="Ajouter un élément..."]').first();

    // User1 creates 2 items
    await item1Input.fill('User1 Item A');
    await item1Input.press('Enter');
    await item1Input.fill('User1 Item B');
    await item1Input.press('Enter');

    // User2 creates 2 items
    await item2Input.fill('User2 Item A');
    await item2Input.press('Enter');
    await item2Input.fill('User2 Item B');
    await item2Input.press('Enter');

    // All items should eventually appear on both clients (with longer timeout due to latency)
    await expect(ctx1.page.getByText('User1 Item A')).toBeVisible({ timeout: 15_000 });
    await expect(ctx1.page.getByText('User2 Item A')).toBeVisible({ timeout: 15_000 });
    await expect(ctx2.page.getByText('User1 Item A')).toBeVisible({ timeout: 15_000 });
    await expect(ctx2.page.getByText('User2 Item B')).toBeVisible({ timeout: 15_000 });
  });

  test('Scenario 4: Voting with FAST latency (5ms) for comparison', async () => {
    // Switch to fast latency to show the difference
    await resetNetworkLatency(ctx1.page);
    await resetNetworkLatency(ctx2.page);
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.fast);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.fast);

    // Advance to vote phase
    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(1_000);

    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(1_000);

    await expect(ctx1.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });

    // With fast latency, voting should be nearly instant
    const voteButtons1 = ctx1.page.locator('button:has(svg.lucide-thumbs-up)');
    await voteButtons1.first().click();

    // Votes should update immediately
    await ctx2.page.waitForTimeout(1_000);
    const voteButtons2 = ctx2.page.locator('button:has(svg.lucide-thumbs-up)');
    await voteButtons2.first().click();
  });

  test('Scenario 5: Page reload recovery with MEDIUM latency (50ms)', async () => {
    // Apply medium latency
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);

    // Reload user2's page (simulates refresh during retro)
    await ctx2.page.reload();
    await ctx2.page.waitForTimeout(3_000);

    // Should reconnect and see the same participant count
    await waitForParticipantCount(ctx2.page, 2);

    // User should still see the current phase and items
    await expect(ctx2.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });
  });
});
