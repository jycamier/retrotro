import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createLeanCoffee, joinRetro, waitForParticipantCount } from './helpers/retro';

let ctx1: { context: BrowserContext; page: Page };
let ctx2: { context: BrowserContext; page: Page };
let lcUrl: string;

test.describe('Lean Coffee session', () => {
  test.describe.configure({ mode: 'serial' });
  test.slow();

  test.beforeAll(async ({ browser }) => {
    ctx1 = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctx2 = await createAuthenticatedContext(browser, DEV_USERS.user1);
  });

  test.afterAll(async () => {
    await ctx1?.context?.close();
    await ctx2?.context?.close();
  });

  test('Create a Lean Coffee and land in waiting room', async () => {
    lcUrl = await createLeanCoffee(ctx1.page);
    expect(lcUrl).toMatch(/\/leancoffee\//);

    // Should see the LC header with Coffee icon and "Lean Coffee" in title area
    await expect(ctx1.page.locator('header')).toBeVisible({ timeout: 10_000 });
    // Should be in waiting phase
    await expect(ctx1.page.getByRole('heading', { name: "Salle d'attente" })).toBeVisible({ timeout: 5_000 });
  });

  test('Second user joins the Lean Coffee', async () => {
    await joinRetro(ctx2.page, lcUrl);
    // Verify user2 sees the waiting room
    await expect(ctx2.page.getByRole('heading', { name: "Salle d'attente" })).toBeVisible({ timeout: 10_000 });
    // Verify WS is connected for both users
    await expect(ctx1.page.getByText('Connected')).toBeVisible({ timeout: 5_000 });
    await expect(ctx2.page.getByText('Connected')).toBeVisible({ timeout: 5_000 });
    // Extra wait for WS room membership to stabilize
    await ctx2.page.waitForTimeout(3_000);
  });

  test('Facilitator advances through icebreaker to propose phase', async () => {
    // waiting → icebreaker
    await ctx1.page.getByRole('button', { name: /Démarrer la rétrospective/i }).click();
    // Wait for ctx2 to receive the phase change
    await expect(ctx2.page.getByText('Icebreaker')).toBeVisible({ timeout: 15_000 });
    await ctx1.page.waitForTimeout(1_000);

    // icebreaker → propose
    await ctx1.page.getByRole('button', { name: /Continuer vers/i }).click();

    // Both users should see the propose phase
    await expect(ctx1.page.getByText('Proposez vos sujets')).toBeVisible({ timeout: 15_000 });
    await expect(ctx2.page.getByText('Proposez vos sujets')).toBeVisible({ timeout: 15_000 });
  });

  test('Users propose topics', async () => {
    // User1 (admin) proposes a topic
    const input1 = ctx1.page.locator('input[placeholder="Proposer un sujet..."]');
    await input1.fill('Améliorer le CI/CD pipeline');
    await input1.press('Enter');
    await expect(ctx1.page.getByText('Améliorer le CI/CD pipeline')).toBeVisible({ timeout: 5_000 });

    // User1 proposes a second topic
    await input1.fill('Revue de la dette technique');
    await input1.press('Enter');
    await expect(ctx1.page.getByText('Revue de la dette technique')).toBeVisible({ timeout: 5_000 });

    // User2 proposes a topic
    const input2 = ctx2.page.locator('input[placeholder="Proposer un sujet..."]');
    await input2.fill('Pair programming sessions');
    await input2.press('Enter');
    await expect(ctx2.page.getByText('Pair programming sessions')).toBeVisible({ timeout: 5_000 });

    // Both users should see all 3 topics
    await expect(ctx1.page.getByText('3 sujets proposés')).toBeVisible({ timeout: 5_000 });
    await expect(ctx2.page.getByText('3 sujets proposés')).toBeVisible({ timeout: 5_000 });
  });

  test('Facilitator advances to vote phase, users vote', async () => {
    // propose → vote
    await ctx1.page.getByRole('button', { name: /Passer au vote/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Both should see vote phase
    await expect(ctx1.page.getByText('Votez pour les sujets')).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText('Votez pour les sujets')).toBeVisible({ timeout: 10_000 });

    // User1 votes for "Améliorer le CI/CD pipeline"
    const addVoteButtons1 = ctx1.page.locator('button[title="Ajouter un vote"]');
    await addVoteButtons1.first().click();
    await ctx1.page.waitForTimeout(500);

    // User2 also votes for a topic
    const addVoteButtons2 = ctx2.page.locator('button[title="Ajouter un vote"]');
    await addVoteButtons2.first().click();
    await ctx2.page.waitForTimeout(500);
  });

  test('Facilitator advances to discuss phase', async () => {
    // vote → discuss
    await ctx1.page.getByRole('button', { name: /Passer à la discussion/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Both should see the discussion view (header badge shows "Discussion")
    await expect(ctx1.page.getByText('À discuter')).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText('À discuter')).toBeVisible({ timeout: 10_000 });

    // Should see the queue and done columns
    await expect(ctx1.page.getByText('À discuter')).toBeVisible({ timeout: 5_000 });
    await expect(ctx1.page.getByText(/Terminé/)).toBeVisible({ timeout: 5_000 });
  });

  test('Facilitator starts discussing first topic', async () => {
    // Click "Sujet suivant" to start discussion
    await ctx1.page.getByRole('button', { name: /Sujet suivant/i }).click();
    await ctx1.page.waitForTimeout(2_000);

    // Current topic should be visible with "En discussion" badge
    await expect(ctx1.page.getByText('En discussion')).toBeVisible({ timeout: 5_000 });

    // User2 should also see the topic being discussed
    await expect(ctx2.page.getByText('En discussion')).toBeVisible({ timeout: 5_000 });
  });

  test('Facilitator creates an action during discussion', async () => {
    // Create an action on the current topic
    const actionInput = ctx1.page.locator('input[placeholder="Titre de l\'action..."]');
    await actionInput.fill('Mettre en place GitHub Actions');
    await ctx1.page.waitForTimeout(200);

    await ctx1.page.getByRole('button', { name: /Créer l'action/i }).click();
    await ctx1.page.waitForTimeout(1_000);

    // Action should be visible (appears in main area and sidebar, use first())
    await expect(ctx1.page.getByText('Mettre en place GitHub Actions').first()).toBeVisible({ timeout: 5_000 });
  });

  test('Facilitator advances to ROTI and ends session', async () => {
    // Move to next topic or directly to ROTI
    // If no more topics in queue, the ROTI button should be visible
    // First try clicking "Sujet suivant" until queue is empty, then go to ROTI
    const nextBtn = ctx1.page.getByRole('button', { name: /Sujet suivant/i });
    while (await nextBtn.isEnabled({ timeout: 1_000 }).catch(() => false)) {
      await nextBtn.click();
      await ctx1.page.waitForTimeout(1_500);
    }

    // Now click ROTI button (or phase_next button)
    const rotiBtn = ctx1.page.getByRole('button', { name: /ROTI|Passer au ROTI/i });
    await rotiBtn.click();
    await ctx1.page.waitForTimeout(2_000);

    // Both should see ROTI phase
    await expect(ctx1.page.getByText('ROTI - Return On Time Invested')).toBeVisible({ timeout: 10_000 });
    await expect(ctx2.page.getByText('ROTI - Return On Time Invested')).toBeVisible({ timeout: 10_000 });

    // Vote ROTI
    await ctx1.page.locator('button').filter({ hasText: '4' }).first().click();
    await ctx1.page.waitForTimeout(500);
    await ctx2.page.locator('button').filter({ hasText: '5' }).first().click();
    await ctx2.page.waitForTimeout(500);

    // Reveal and end
    await ctx1.page.getByRole('button', { name: /Révéler les résultats/i }).click();
    await ctx1.page.waitForTimeout(1_000);
    await ctx1.page.getByRole('button', { name: /Terminer la rétrospective/i }).click();
    await ctx1.page.getByRole('button', { name: /Confirmer/i }).click();
    await ctx1.page.waitForTimeout(3_000);

    // Summary should appear
    await expect(ctx1.page.getByText(/Rétrospective terminée/i)).toBeVisible({ timeout: 10_000 });
  });
});
