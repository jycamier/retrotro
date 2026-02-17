import { test, expect, type Page, type BrowserContext } from '@playwright/test';
import { DEV_USERS, createAuthenticatedContext } from './helpers/auth';
import { createTeamAndRetro, joinRetro, waitForParticipantCount, nextPhase } from './helpers/retro';

/**
 * Test for grouping bug: https://github.com/jycamier/retrotro/issues/XXX
 * 
 * BUG DESCRIPTION:
 * When dragging an item that already has child items grouped under it to a new parent,
 * the child items become orphaned with stale groupId references.
 * 
 * SCENARIO:
 * 1. Create 4 items: Item1, Item2, Item3, Item4
 * 2. Group Item1 under Item2 → Item1.groupId = Item2.id
 * 3. Re-group Item2 (with Item1) under Item3
 *    - BUG: Item1 is lost because Item1.groupId still points to Item2
 *    - FIX: Item1 should move with Item2 to be grouped under Item3
 * 
 * EXPECTED BEHAVIOR AFTER FIX:
 * - Item3 has both Item1 and Item2 grouped under it
 * - Item1 and Item2 both correctly point to Item3 as their groupId
 */

let ctxAdmin: { context: BrowserContext; page: Page };
let ctxUser1: { context: BrowserContext; page: Page };
let retroUrl: string;

test.describe('Grouping bug - re-grouping items loses previously grouped items', () => {
  test.describe.configure({ mode: 'serial' });

  test.beforeAll(async ({ browser }) => {
    ctxAdmin = await createAuthenticatedContext(browser, DEV_USERS.admin);
    ctxUser1 = await createAuthenticatedContext(browser, DEV_USERS.user1);
  });

  test.afterAll(async () => {
    await ctxAdmin?.context?.close();
    await ctxUser1?.context?.close();
  });

  test('Setup: Create retro and reach grouping phase with items', async () => {
    // Admin creates retro
    retroUrl = await createTeamAndRetro(ctxAdmin.page);

    // User1 joins
    await joinRetro(ctxUser1.page, retroUrl);

    // Both users see 2 participants
    await waitForParticipantCount(ctxAdmin.page, 2);
    await waitForParticipantCount(ctxUser1.page, 2);

    // Start retro (from waiting → icebreaker)
    await ctxAdmin.page.getByRole('button', { name: /Démarrer la rétrospective/i }).click();
    await ctxAdmin.page.waitForTimeout(1_000);

    // Skip icebreaker → brainstorm
    await ctxAdmin.page.getByRole('button', { name: /Continuer vers Brainstorm/i }).click();
    await ctxAdmin.page.waitForTimeout(1_000);

    // Wait for brainstorm phase
    await expect(ctxAdmin.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 10_000 });
    await expect(ctxUser1.page.getByText(/brainstorm/i)).toBeVisible({ timeout: 10_000 });

    // Create 4 items in Went Well column
    const getItemInput = (page: Page) => page.locator('input[placeholder="Ajouter un élément..."]').first();

    // Item 1
    await getItemInput(ctxAdmin.page).fill('Item 1');
    await getItemInput(ctxAdmin.page).press('Enter');
    await ctxAdmin.page.waitForTimeout(500);

    // Item 2
    await getItemInput(ctxAdmin.page).fill('Item 2');
    await getItemInput(ctxAdmin.page).press('Enter');
    await ctxAdmin.page.waitForTimeout(500);

    // Item 3
    await getItemInput(ctxAdmin.page).fill('Item 3');
    await getItemInput(ctxAdmin.page).press('Enter');
    await ctxAdmin.page.waitForTimeout(500);

    // Item 4
    await getItemInput(ctxAdmin.page).fill('Item 4');
    await getItemInput(ctxAdmin.page).press('Enter');
    await ctxAdmin.page.waitForTimeout(1_000);

    // Verify all 4 items are visible
    await expect(ctxAdmin.page.getByText('Item 1')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Item 2')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Item 3')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Item 4')).toBeVisible();

    // Move to grouping phase
    await nextPhase(ctxAdmin.page); // brainstorm → group
    await ctxAdmin.page.waitForTimeout(2_000);

    await expect(ctxAdmin.page.getByText(/groupage/i)).toBeVisible({ timeout: 10_000 });
    await expect(ctxUser1.page.getByText(/groupage/i)).toBeVisible({ timeout: 10_000 });
  });

  test('Bug: Drag Item 1 to Item 2 to group them', async () => {
    // Both items should be visible in group phase
    await expect(ctxAdmin.page.getByText('Item 1')).toBeVisible();
    await expect(ctxAdmin.page.getByText('Item 2')).toBeVisible();

    // Get drag handles
    const dragHandles = ctxAdmin.page.locator('[class*="rotate-90"]'); // GripVertical rotated

    // Drag Item 1 onto Item 2
    const item1Handle = dragHandles.first();
    const item2Text = ctxAdmin.page.getByText('Item 2').first();

    await item1Handle.dragTo(item2Text);
    await ctxAdmin.page.waitForTimeout(1_000);

    // Verify items are grouped - click on grouping indicator
    const groupedIndicator = ctxAdmin.page.getByText(/1 item groupé/);
    await groupedIndicator.waitFor({ timeout: 5_000 });
    await groupedIndicator.click();
    await ctxAdmin.page.waitForTimeout(500);

    // Verify Item 1 appears in the grouped items section
    await expect(ctxAdmin.page.getByText('Item 1')).toBeVisible();
    expect(await ctxAdmin.page.getByText('Item 1').count()).toBeGreaterThanOrEqual(1);

    // Also verify on User1's page
    const groupedIndicator2 = ctxUser1.page.getByText(/1 item groupé/);
    await groupedIndicator2.waitFor({ timeout: 5_000 });
    await groupedIndicator2.click();
    await ctxUser1.page.waitForTimeout(500);

    await expect(ctxUser1.page.getByText('Item 1')).toBeVisible();
  });

  test('Bug: Drag Item 2 (which now has Item 1 grouped) to Item 3 - THIS LOSES ITEM 1', async () => {
    // Verify current state before re-grouping
    // Item 2 should have Item 1 grouped under it
    let groupedIndicator = ctxAdmin.page.getByText(/1 item groupé/);
    await groupedIndicator.waitFor({ timeout: 5_000 });
    await groupedIndicator.click();
    await ctxAdmin.page.waitForTimeout(500);

    // Verify Item 1 is visible before the re-group
    await expect(ctxAdmin.page.getByText('Item 1')).toBeVisible();

    // Now collapse it to prepare for re-grouping
    groupedIndicator = ctxAdmin.page.getByText(/1 item groupé/);
    await groupedIndicator.click();
    await ctxAdmin.page.waitForTimeout(500);

    // Get all drag handles
    const dragHandles = ctxAdmin.page.locator('[class*="rotate-90"]');
    
    // Find which handle corresponds to Item 2
    // Item 2 should be the 2nd item in the list (Item 1 was grouped)
    const item2Handle = dragHandles.nth(1);
    const item3Text = ctxAdmin.page.getByText('Item 3').first();

    // Drag Item 2 to Item 3
    await item2Handle.dragTo(item3Text);
    await ctxAdmin.page.waitForTimeout(2_000);

    // BUG: Check if Item 1 is still grouped under Item 2
    // It should show "2 items groupés" (Item 1 + Item 2 under Item 3)
    // But the bug causes Item 1 to be lost

    const groupedIndicatorAfter = ctxAdmin.page.getByText(/items groupé/);
    await groupedIndicatorAfter.waitFor({ timeout: 5_000 });
    const groupedText = await groupedIndicatorAfter.textContent();

    console.log('Grouped indicator text after re-group:', groupedText);

    // Click to expand and see what's actually grouped
    await groupedIndicatorAfter.click();
    await ctxAdmin.page.waitForTimeout(500);

    // Try to find Item 1
    const item1Elements = ctxAdmin.page.getByText('Item 1');
    const item1Count = await item1Elements.count();

    console.log('Item 1 count after re-grouping:', item1Count);

    // This should show Item 1 is still visible, but it won't if the bug is present
    if (item1Count === 0) {
      console.log('BUG REPRODUCED: Item 1 was lost when Item 2 was re-grouped!');
    }
    
    // Verify on User1's page as well
    const groupedIndicator2 = ctxUser1.page.getByText(/items groupé/);
    await groupedIndicator2.waitFor({ timeout: 5_000 });
    await groupedIndicator2.click();
    await ctxUser1.page.waitForTimeout(500);

    const item1Elements2 = ctxUser1.page.getByText('Item 1');
    const item1Count2 = await item1Elements2.count();
    console.log('Item 1 count on User1 page after re-grouping:', item1Count2);
  });

  test('Verify: Item 1 should still be grouped after the re-grouping operation', async () => {
    // Expand all grouped items to check
    const groupedIndicators = ctxAdmin.page.getByText(/items groupé/);
    const count = await groupedIndicators.count();

    console.log('Total grouped indicators:', count);

    // For each group, expand and check for Item 1
    for (let i = 0; i < count; i++) {
      const indicator = groupedIndicators.nth(i);
      await indicator.click();
      await ctxAdmin.page.waitForTimeout(300);
    }

    // Search for Item 1 - it should be somewhere in the grouped items
    const item1All = ctxAdmin.page.getByText('Item 1');
    const item1Count = await item1All.count();

    console.log('Final Item 1 count:', item1Count);

    if (item1Count === 0) {
      console.log('FAILED: Item 1 is completely missing from the retro!');
    } else {
      console.log('PASSED: Item 1 still exists in the retro');
    }

    // This assertion should pass but will fail if bug exists
    expect(item1Count).toBeGreaterThan(0, 'Item 1 should still exist after re-grouping');
  });
});
