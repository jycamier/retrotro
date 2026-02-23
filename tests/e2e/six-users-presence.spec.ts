import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro } from './helpers/retro';

/**
 * Set a page offline/online via Chrome DevTools Protocol.
 * This kills the WebSocket immediately (simulating a network drop).
 */
async function setOffline(page: Page, offline: boolean) {
  const client = await page.context().newCDPSession(page);
  await client.send('Network.emulateNetworkConditions', {
    offline,
    downloadThroughput: offline ? 0 : -1,
    uploadThroughput: offline ? 0 : -1,
    latency: 0,
  });
  await client.detach();
}

const ALL_USERS = [
  DEV_USERS.admin,
  DEV_USERS.manager,
  DEV_USERS.facilitator,
  DEV_USERS.user1,
  DEV_USERS.user2,
  DEV_USERS.user3,
] as const;

let contexts: { context: BrowserContext; page: Page }[];
let retroUrl: string;

test.describe('Six users presence in waiting room', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    // Create 6 isolated browser contexts, one per user
    contexts = [];
    for (const user of ALL_USERS) {
      contexts.push(await createAuthenticatedContext(browser, user));
    }
  });

  test.afterAll(async () => {
    for (const ctx of contexts ?? []) {
      await ctx?.context?.close();
    }
  });

  test('Users join one by one and presence count increments correctly', async () => {
    const total = contexts.length; // 6

    // Admin creates the retro — first user in the waiting room
    retroUrl = await createTeamAndRetro(contexts[0].page);

    // Admin should see 1/6
    await expect(async () => {
      const text = await contexts[0].page.textContent('body');
      expect(text).toContain(`1/${total}`);
    }).toPass({ timeout: 15_000 });

    // Users join one by one; after each join, verify the count for everyone already present
    for (let joining = 1; joining < total; joining++) {
      await joinRetro(contexts[joining].page, retroUrl);

      const expected = `${joining + 1}/${total}`;

      // Every user already in the room (0..joining) should see the updated count
      for (let i = 0; i <= joining; i++) {
        await expect(async () => {
          const text = await contexts[i].page.textContent('body');
          expect(text).toContain(expected);
        }).toPass({ timeout: 15_000 });
      }
    }

    // Final state: all 6 see 6/6 and the team members grid
    for (let i = 0; i < total; i++) {
      await expect(contexts[i].page.getByText("Membres de l'équipe")).toBeVisible({ timeout: 5_000 });
    }
    // Facilitator sees "Tous les membres sont connectés !"
    await expect(contexts[0].page.getByText('Tous les membres sont connectés')).toBeVisible({ timeout: 5_000 });
  });

  test('After one user reloads, all 6 still shown as connected', async () => {
    // User3 (index 2) reloads the page
    await contexts[2].page.reload();
    await contexts[2].page.waitForTimeout(3_000);

    // Wait for state to stabilise (grace period is 10s, reconnection should be < 3s)
    // All 6 users should still see 6/6
    for (let i = 0; i < contexts.length; i++) {
      await expect(async () => {
        const text = await contexts[i].page.textContent('body');
        expect(text).toContain('6/6');
      }).toPass({ timeout: 15_000 });
    }
  });

  test('User reconnects just before grace period expires — race condition test', async () => {
    // This test targets the exact race condition:
    // 1. User disconnects → backend starts 10s grace period timer
    // 2. User reconnects at ~9s → JoinRoom is called while the timer is still pending
    // 3. WITHOUT the fix: JoinRoom doesn't cancel the pending timer,
    //    the timer fires ~1s later and broadcasts participant_left AFTER the user rejoined
    // 4. WITH the fix: JoinRoom cancels the pending timer, user stays visible

    // Navigate user (index 3) away — WebSocket closes, grace period starts
    await contexts[3].page.goto('about:blank');

    // Wait 9s — just before the 10s grace period expires
    await contexts[0].page.waitForTimeout(9_000);

    // User comes back just before the timer fires
    await contexts[3].page.goto(retroUrl);

    // Wait long enough for the grace period timer to have fired (if not canceled)
    // If the fix is missing, the timer fires ~1s after rejoin → participant_left
    await contexts[0].page.waitForTimeout(5_000);

    // All 6 should still see 6/6
    // WITHOUT the fix: admin will see 5/6 because the timer fired after the rejoin
    for (let i = 0; i < contexts.length; i++) {
      await expect(async () => {
        const text = await contexts[i].page.textContent('body');
        expect(text).toContain('6/6');
      }).toPass({ timeout: 15_000 });
    }
  });

  test('After multiple users reload simultaneously, all 6 still connected', async () => {
    // Users 1, 3, 5 (indices 1, 3, 5) reload at the same time
    await Promise.all([
      contexts[1].page.reload(),
      contexts[3].page.reload(),
      contexts[5].page.reload(),
    ]);

    // Wait for reconnections
    await contexts[0].page.waitForTimeout(5_000);

    // All pages should show 6/6
    for (let i = 0; i < contexts.length; i++) {
      await expect(async () => {
        const text = await contexts[i].page.textContent('body');
        expect(text).toContain('6/6');
      }).toPass({ timeout: 20_000 });
    }
  });

  test('After ALL 6 users reload, everyone reconnects and sees 6/6', async () => {
    // Everyone reloads at once — worst case scenario
    await Promise.all(
      contexts.map(ctx => ctx.page.reload())
    );

    // Wait for all reconnections
    await contexts[0].page.waitForTimeout(5_000);

    for (let i = 0; i < contexts.length; i++) {
      await expect(async () => {
        const text = await contexts[i].page.textContent('body');
        expect(text).toContain('6/6');
      }).toPass({ timeout: 20_000 });
    }
  });
});
