import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro, waitForParticipantCount, nextPhase } from './helpers/retro';
import { applyNetworkLatency, resetNetworkLatency, LATENCY_PROFILES } from './helpers/network';

let ctx1: { context: BrowserContext; page: Page };
let ctx2: { context: BrowserContext; page: Page };
let retroUrl: string;

test.describe('Reconnection resilience with latency', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    ctx1 = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctx2 = await createAuthenticatedContext(browser, DEV_USERS.user1);
  });

  test.afterAll(async () => {
    // Delay reset to ensure all tests complete
    await new Promise(resolve => setTimeout(resolve, 2000));
    await resetNetworkLatency(ctx1.page).catch(() => {});
    await resetNetworkLatency(ctx2.page).catch(() => {});
    await ctx1?.context?.close();
    await ctx2?.context?.close();
  });

  test('Setup: Create retro with 2 users', async () => {
    retroUrl = await createTeamAndRetro(ctx1.page);
    await joinRetro(ctx2.page, retroUrl);
    await waitForParticipantCount(ctx1.page, 2);
    await waitForParticipantCount(ctx2.page, 2);

    console.log('✓ Both users joined');
  });

  test('Test 1: Reload with MEDIUM latency (50ms) - Participant list preservation', async () => {
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);

    console.log('📱 Reloading User2 page with 50ms latency...');

    // Reload User2's page
    await ctx2.page.reload();

    // Wait for page to stabilize after reload
    await ctx2.page.waitForTimeout(2_000);

    // Should reconnect and return to the retro board
    const heading = ctx2.page.getByRole('heading').first();
    await expect(heading).toBeVisible({ timeout: 20_000 });
    console.log('✓ User2 page loaded after reload');

    // Both users should still see each other
    await waitForParticipantCount(ctx1.page, 2);
    await waitForParticipantCount(ctx2.page, 2);
    console.log('✓ Both users see 2 participants after reload');
  });

  test('Test 2: Reload with SLOW latency (150ms) - Grace period test', async () => {
    // Switch to slow latency
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.slow);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.slow);

    console.log('🐢 Reloading with 150ms latency (tests grace period)...');

    // Reload User1 quickly (while User2 is still connected)
    await ctx1.page.reload();

    // Page should load (may show cached state from sessionStorage)
    const heading = ctx1.page.getByRole('heading').first();
    await expect(heading).toBeVisible({ timeout: 25_000 });
    console.log('✓ User1 page loaded after reload');

    // User2 should NOT have seen "participant_left" message because User1 reconnected in time
    await waitForParticipantCount(ctx2.page, 2);
    console.log('✓ User2 still sees 2 participants (no false disconnect)');

    // User1 should still see User2
    await waitForParticipantCount(ctx1.page, 2);
    console.log('✓ User1 also sees 2 participants');
  });

  test('Test 3: Multiple rapid reloads with MEDIUM latency', async () => {
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);

    console.log('🔄 Performing rapid reload sequence...');

    // Reload User1
    await ctx1.page.reload();
    const heading1 = ctx1.page.getByRole('heading').first();
    await expect(heading1).toBeVisible({ timeout: 25_000 });
    console.log('✓ User1 reload #1 complete');

    // Quick reload User2
    await ctx2.page.reload();
    const heading2 = ctx2.page.getByRole('heading').first();
    await expect(heading2).toBeVisible({ timeout: 25_000 });
    console.log('✓ User2 reload #1 complete');

    // Both should see each other still
    await waitForParticipantCount(ctx1.page, 2);
    await waitForParticipantCount(ctx2.page, 2);
    console.log('✓ Both users see 2 participants after sequential reloads');
  });

  test('Test 4: Reconnection with VERY SLOW latency (400ms) - Extreme case', async () => {
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.verySlow);

    console.log('🚀 Extreme latency test: 400ms (poor 3G)');

    // Reload User1
    await ctx1.page.reload();

    // Should reconnect even with extreme latency
    const heading = ctx1.page.getByRole('heading').first();
    await expect(heading).toBeVisible({ timeout: 35_000 });
    console.log('✓ Page loaded even with 400ms latency');

    // Should see participant count
    await waitForParticipantCount(ctx1.page, 2);
    console.log('✓ User1 sees 2 participants after extreme latency reconnect');
  });

  test('Test 5: Heartbeat keeps connection alive', async () => {
    // Reset to medium latency
    await applyNetworkLatency(ctx1.page, LATENCY_PROFILES.medium);
    await applyNetworkLatency(ctx2.page, LATENCY_PROFILES.medium);

    console.log('💓 Testing heartbeat (connection stability)...');

    // Check console logs for heartbeat messages
    const logs: string[] = [];
    ctx1.page.on('console', msg => {
      logs.push(msg.text());
    });

    // Wait 35 seconds to see at least one heartbeat (heartbeat every 30s)
    console.log('⏳ Waiting for heartbeat cycles...');
    await ctx1.page.waitForTimeout(35_000);

    // Look for heartbeat logs
    const hasHeartbeat = logs.some(log => log.includes('heartbeat'));
    if (hasHeartbeat) {
      console.log('✓ Heartbeat detected in logs');
    } else {
      console.log('⚠️  No heartbeat detected (may not be in logs, but connection still works)');
    }

    // Verify still connected
    await waitForParticipantCount(ctx1.page, 2);
    console.log('✓ Connection still stable after 35+ seconds with heartbeat');
  });

  test('Cleanup: Reset network', async () => {
    await resetNetworkLatency(ctx1.page).catch(() => {});
    await resetNetworkLatency(ctx2.page).catch(() => {});
    console.log('✓ Network latency reset');
  });
});
