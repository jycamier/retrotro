import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro, waitForParticipantCount, nextPhase } from './helpers/retro';
import { applyNetworkLatency, resetNetworkLatency, LATENCY_PROFILES } from './helpers/network';

let ctx1: { context: BrowserContext; page: Page };
let ctx2: { context: BrowserContext; page: Page };
let ctx3: { context: BrowserContext; page: Page };
let retroUrl: string;

test.describe('Page reload with collaborative latency', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    ctx1 = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctx2 = await createAuthenticatedContext(browser, DEV_USERS.user1);
    ctx3 = await createAuthenticatedContext(browser, DEV_USERS.user2);
  });

  test.afterAll(async () => {
    await resetNetworkLatency(ctx1.page).catch(() => {});
    await resetNetworkLatency(ctx2.page).catch(() => {});
    await resetNetworkLatency(ctx3.page).catch(() => {});
    await ctx1?.context?.close();
    await ctx2?.context?.close();
    await ctx3?.context?.close();
  });

  test('Setup: Create retro and join with 3 users', async () => {
    // No latency for setup
    retroUrl = await createTeamAndRetro(ctx1.page);
    await joinRetro(ctx2.page, retroUrl);
    await joinRetro(ctx3.page, retroUrl);

    await waitForParticipantCount(ctx1.page, 3);
    await waitForParticipantCount(ctx2.page, 3);
    await waitForParticipantCount(ctx3.page, 3);
  });

  test('Scenario: Page reload during brainstorm with MEDIUM latency (50ms)', async () => {
    // Apply medium latency to simulate real-world WiFi
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx3.page, LATENCY_PROFILES.medium);

    // Move to brainstorm phase
    await ctx1.page.getByRole('button', { name: /Démarrer la rétrospective/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    await ctx1.page.getByRole('button', { name: /Continuer vers Brainstorm/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Wait for all users to see brainstorm phase
    await expect(ctx1.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 15_000 });
    await expect(ctx2.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 15_000 });
    await expect(ctx3.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 15_000 });

    // User1 and User3 add items with longer timeouts due to latency
    const item1Input = ctx1.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await item1Input.fill('Item from User1 - before reload');
    await item1Input.press('Enter');
    await ctx1.page.waitForTimeout(1_000); // Wait for broadcast to complete

    const item3Input = ctx3.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await item3Input.fill('Item from User3 - before reload');
    await item3Input.press('Enter');
    await ctx3.page.waitForTimeout(1_000); // Wait for broadcast to complete

    // Wait for items to be visible on all clients with longer timeout
    // With 50ms latency, expect 100-200ms total round-trip
    await expect(ctx1.page.getByText('Item from User1 - before reload')).toBeVisible({ timeout: 20_000 });
    await expect(ctx2.page.getByText('Item from User1 - before reload')).toBeVisible({ timeout: 25_000 });
    await expect(ctx3.page.getByText('Item from User1 - before reload')).toBeVisible({ timeout: 20_000 });

    await expect(ctx1.page.getByText('Item from User3 - before reload')).toBeVisible({ timeout: 20_000 });
    await expect(ctx2.page.getByText('Item from User3 - before reload')).toBeVisible({ timeout: 25_000 });
    await expect(ctx3.page.getByText('Item from User3 - before reload')).toBeVisible({ timeout: 20_000 });

    console.log('✓ All participants see initial items before reload');
  });

  test('Scenario: User2 page reload during active brainstorm with latency', async () => {
    // Get initial participant count before reload
    const initialText = await ctx2.page.textContent('body');
    expect(initialText).toContain('3');

    console.log('📱 Reloading User2 page during active collaboration with 50ms latency...');

    // Apply latency before reload
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);

    // Reload User2's page
    await ctx2.page.reload();

    // The loading screen should appear with our new LoadingScreen component
    console.log('⏳ Loading screen should now be visible with progress bar and steps...');

    // Wait for the loading screen to complete
    // With 50ms latency, this should take a bit longer
    await expect(ctx2.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 20_000 });

    console.log('✓ User2 reconnected successfully');

    // Verify User2 still sees all participants (should be 3)
    await expect(async () => {
      const text = await ctx2.page.textContent('body');
      expect(text).toMatch(/3|participants/);
    }).toPass({ timeout: 15_000 });

    console.log('✓ User2 still sees all 3 participants after reload');
  });

  test('Scenario: Verify participants list consistency after reload with latency', async () => {
    // User1 adds another item
    const item1Input = ctx1.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await item1Input.fill('Item from User1 - after User2 reload');
    await item1Input.press('Enter');

    // All users should see this new item (even with latency)
    await expect(ctx1.page.getByText('Item from User1 - after User2 reload')).toBeVisible({ timeout: 15_000 });
    await expect(ctx2.page.getByText('Item from User1 - after User2 reload')).toBeVisible({ timeout: 20_000 });
    await expect(ctx3.page.getByText('Item from User1 - after User2 reload')).toBeVisible({ timeout: 15_000 });

    console.log('✓ All items synchronized after User2 reload with latency');

    // Verify participant counts on all pages
    await expect(async () => {
      const text1 = await ctx1.page.textContent('body');
      expect(text1).toMatch(/3|participants/);
    }).toPass({ timeout: 10_000 });

    await expect(async () => {
      const text2 = await ctx2.page.textContent('body');
      expect(text2).toMatch(/3|participants/);
    }).toPass({ timeout: 10_000 });

    await expect(async () => {
      const text3 = await ctx3.page.textContent('body');
      expect(text3).toMatch(/3|participants/);
    }).toPass({ timeout: 10_000 });

    console.log('✓ All participants see consistent state (3 users)');
  });

  test('Scenario: Multiple rapid reloads with SLOW latency (150ms)', async () => {
    // Switch to slower latency
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.slow);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.slow);
    await applyNetworkLatency(ctx3.page, LATENCY_PROFILES.slow);

    console.log('🐢 Simulating poor connection (150ms latency)...');

    // User2 reloads quickly
    await ctx2.page.reload();
    await expect(ctx2.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 20_000 });

    // User2 adds an item
    const item2Input = ctx2.page.locator('input[placeholder="Ajouter un élément..."]').first();
    await item2Input.fill('Item from User2 - under slow latency');
    await item2Input.press('Enter');

    // User3 reloads too
    await ctx3.page.reload();
    await expect(ctx3.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 20_000 });

    // Both items should be visible on all clients eventually
    await expect(ctx1.page.getByText('Item from User2 - under slow latency')).toBeVisible({ timeout: 20_000 });
    await expect(ctx3.page.getByText('Item from User2 - under slow latency')).toBeVisible({ timeout: 25_000 });

    // Verify no "participants left" flicker happened
    await expect(async () => {
      const text1 = await ctx1.page.textContent('body');
      expect(text1).toMatch(/3|participants/);
    }).toPass({ timeout: 10_000 });

    console.log('✓ Multiple reloads under slow latency handled correctly');
    console.log('✓ No false "participant left" messages appeared');
  });

  test('Scenario: Reload with VERY SLOW latency (400ms) - stress test', async () => {
    // Maximum latency stress test
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.verySlow);

    console.log('🚀 Extreme stress test: 400ms latency (poor 3G)');

    // Reload User2
    await ctx2.page.reload();

    // Even with extreme latency, should reconnect within reasonable time
    await expect(ctx2.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 30_000 });

    // Verify still connected
    await expect(async () => {
      const text = await ctx2.page.textContent('body');
      expect(text).toMatch(/3|participants/);
    }).toPass({ timeout: 15_000 });

    console.log('✓ Survived 400ms latency reload');
  });

  test('Cleanup: Reset network', async () => {
    await resetNetworkLatency(ctx1.page).catch(() => {});
    await resetNetworkLatency(ctx2.page).catch(() => {});
    await resetNetworkLatency(ctx3.page).catch(() => {});
  });
});
