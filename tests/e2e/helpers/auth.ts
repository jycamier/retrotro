import { type Browser, type BrowserContext, type Page } from '@playwright/test';

export interface DevUser {
  email: string;
  displayName: string;
}

export const DEV_USERS = {
  admin: { email: 'admin@retrotro.dev', displayName: 'Dev Admin' },
  user1: { email: 'user1@retrotro.dev', displayName: 'User One' },
  user2: { email: 'user2@retrotro.dev', displayName: 'User Two' },
} as const;

/**
 * Log in as a dev user by clicking their card on the login page.
 */
export async function devLogin(page: Page, user: DevUser): Promise<void> {
  await page.goto('/login');
  // Wait for dev user cards to load
  await page.waitForSelector('text=' + user.displayName, { timeout: 10_000 });
  await page.getByText(user.displayName).click();
  // Wait for redirect to dashboard
  await page.waitForURL('/', { timeout: 10_000 });
}

/**
 * Create an isolated browser context with a logged-in dev user.
 * Uses ?session= param for storage isolation.
 */
export async function createAuthenticatedContext(
  browser: Browser,
  user: DevUser,
): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext();
  const page = await context.newPage();

  await devLogin(page, user);

  return { context, page };
}
