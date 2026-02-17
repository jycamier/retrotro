import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, nextPhase } from './helpers/retro';

let ctx: { context: BrowserContext; page: Page };

test.describe('Team Actions Kanban', () => {
  test.slow();

  test.beforeAll(async ({ browser }) => {
    ctx = await createAuthenticatedContext(browser, DEV_USERS.admin);
  });

  test.afterAll(async () => {
    await ctx?.context?.close();
  });

  test('actions from completed retro appear in kanban', async () => {
    const page = ctx.page;

    // 1. Create retro (lands in waiting room)
    await createTeamAndRetro(page);

    // 2. Start retro: waiting → icebreaker
    await nextPhase(page);
    await page.waitForTimeout(1_500);

    // 3. Skip icebreaker → brainstorm
    await nextPhase(page);
    await page.waitForTimeout(1_500);

    // 4. Add an item in brainstorm
    const itemInput = page.locator('input[placeholder="Ajouter un élément..."]').first();
    await itemInput.fill('Test item for kanban');
    await itemInput.press('Enter');
    await page.waitForTimeout(500);

    // 5. Advance: brainstorm → group → vote → discuss → action
    for (let i = 0; i < 4; i++) {
      await nextPhase(page);
      await page.waitForTimeout(1_500);
    }

    // 6. Verify we're in action phase
    await expect(page.getByRole('heading', { name: 'Actions' })).toBeVisible({ timeout: 10_000 });

    // 7. Create an action
    const actionInput = page.locator('input[placeholder*="action"]');
    await actionInput.fill('Deploy kanban feature');
    await page.getByRole('button', { name: /Créer l'action/i }).click();
    await page.waitForTimeout(500);

    await expect(page.getByText('Deploy kanban feature')).toBeVisible({ timeout: 5_000 });

    // 8. End the retro (action → roti → end)
    await nextPhase(page); // action → roti
    await page.waitForTimeout(2_000);

    // Vote on ROTI (click rating 4)
    await page.locator('button').filter({ hasText: '4' }).first().click();
    await page.waitForTimeout(1_000);

    // Reveal results then end retro
    await page.getByRole('button', { name: /Révéler les résultats/i }).click();
    await page.waitForTimeout(1_000);
    await page.getByRole('button', { name: /Terminer la rétrospective/i }).click();
    await page.getByRole('button', { name: /Confirmer/i }).click();
    await page.waitForTimeout(3_000);

    // 9. Navigate to team actions kanban
    await page.goto('/');
    await page.getByText('Dev Team').click();
    await page.waitForURL(/\/teams\//);
    await page.getByRole('link', { name: 'Actions' }).click();
    await page.waitForTimeout(2_000);

    // 10. Verify kanban board structure
    await expect(page.getByRole('heading', { name: 'À faire' })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('heading', { name: 'En cours' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Terminé' })).toBeVisible();

    // 11. Verify our action is in "À faire" column
    const todoColumn = page.locator('[data-testid="column-todo"]');
    await expect(todoColumn.getByText('Deploy kanban feature')).toBeVisible({ timeout: 10_000 });

    // 12. Drag action from "À faire" to "En cours" using pointer events for @dnd-kit
    const card = todoColumn.getByText('Deploy kanban feature');
    const inProgressColumn = page.locator('[data-testid="column-in_progress"]');

    const cardBox = await card.boundingBox();
    const targetBox = await inProgressColumn.boundingBox();

    if (cardBox && targetBox) {
      const startX = cardBox.x + cardBox.width / 2;
      const startY = cardBox.y + cardBox.height / 2;
      const endX = targetBox.x + targetBox.width / 2;
      const endY = targetBox.y + targetBox.height / 2;

      await page.mouse.move(startX, startY);
      await page.mouse.down();
      // Move gradually to trigger @dnd-kit's activation distance (8px)
      await page.mouse.move(startX + 10, startY, { steps: 5 });
      await page.waitForTimeout(100);
      await page.mouse.move(endX, endY, { steps: 20 });
      await page.waitForTimeout(200);
      await page.mouse.up();
    }
    await page.waitForTimeout(2_000);

    // 13. Verify card moved to "En cours"
    await expect(inProgressColumn.getByText('Deploy kanban feature')).toBeVisible({ timeout: 5_000 });
  });
});
