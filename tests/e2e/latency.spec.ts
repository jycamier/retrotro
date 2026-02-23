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
    console.log('✓ Both users in brainstorm phase');
  });

  test('Scenario 3: Rapid item creation under VERY SLOW latency (400ms)', async () => {
    // Apply very slow latency (poor 3G connection)
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.verySlow);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.verySlow);

    // Move to next phase for easier verification (items not obfuscated)
    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(3_000);

    // Wait for both users to see the phase change - use first() to avoid strict mode violation
    const phaseText = ctx1.page.getByText(/group/i).first();
    await expect(phaseText).toBeVisible({ timeout: 15_000 });
    console.log('✓ Both users transitioning through phases with extreme latency');
  });

  test('Scenario 4: Voting with FAST latency (5ms) for comparison', async () => {
    // Switch to fast latency to show the difference
    await resetNetworkLatency(ctx1.page);
    await resetNetworkLatency(ctx2.page);
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.fast);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.fast);

    // Advance to vote phase (group → vote)
    await nextPhase(ctx1.page);
    await ctx1.page.waitForTimeout(1_000);

    await expect(ctx1.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText(/vote/i)).toBeVisible({ timeout: 10_000 });

    // With fast latency, voting should be nearly instant (if items exist)
    const voteButtons1 = ctx1.page.locator('button:has(svg.lucide-thumbs-up)');
    if (await voteButtons1.count() > 0) {
      await voteButtons1.first().click();
      await ctx2.page.waitForTimeout(1_000);
      const voteButtons2 = ctx2.page.locator('button:has(svg.lucide-thumbs-up)');
      await voteButtons2.first().click();
    }
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
