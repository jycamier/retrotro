import { Page } from '@playwright/test';

export interface LatencyConfig {
  name: string;
  downloadKbps: number;
  uploadKbps: number;
  latencyMs: number;
}

/**
 * Network latency configurations for different scenarios
 */
export const LATENCY_PROFILES = {
  // Very fast network (localhost/LAN)
  fast: {
    name: 'Fast Network (LAN)',
    downloadKbps: 100000,
    uploadKbps: 100000,
    latencyMs: 5,
  } as LatencyConfig,

  // Medium network (typical home wifi)
  medium: {
    name: 'Medium Network (WiFi)',
    downloadKbps: 10000,
    uploadKbps: 5000,
    latencyMs: 50,
  } as LatencyConfig,

  // Slow network (4G/mobile)
  slow: {
    name: 'Slow Network (4G)',
    downloadKbps: 1000,
    uploadKbps: 500,
    latencyMs: 150,
  } as LatencyConfig,

  // Very slow network (3G/poor connection)
  verySlow: {
    name: 'Very Slow Network (3G)',
    downloadKbps: 400,
    uploadKbps: 200,
    latencyMs: 400,
  } as LatencyConfig,
} as const;

/**
 * Apply network throttling to a page using Chrome DevTools Protocol
 * This simulates real network latency and bandwidth limitations
 */
export async function applyNetworkLatency(
  page: Page,
  latencyConfig: LatencyConfig,
): Promise<void> {
  // Get the CDP session for low-level network control
  const client = await page.context().newCDPSession(page);

  try {
    await client.send('Network.emulateNetworkConditions', {
      offline: false,
      downloadThroughput: (latencyConfig.downloadKbps * 1024) / 8, // Convert to bytes/sec
      uploadThroughput: (latencyConfig.uploadKbps * 1024) / 8, // Convert to bytes/sec
      latency: latencyConfig.latencyMs,
    });

    console.log(`Applied network latency: ${latencyConfig.name}`);
    console.log(`  - Latency: ${latencyConfig.latencyMs}ms`);
    console.log(`  - Download: ${latencyConfig.downloadKbps} kbps`);
    console.log(`  - Upload: ${latencyConfig.uploadKbps} kbps`);
  } catch (error) {
    console.error('Failed to apply network latency:', error);
    throw error;
  } finally {
    await client.detach();
  }
}

/**
 * Reset network to normal conditions
 */
export async function resetNetworkLatency(page: Page): Promise<void> {
  const client = await page.context().newCDPSession(page);

  try {
    await client.send('Network.emulateNetworkConditions', {
      offline: false,
      downloadThroughput: -1, // No throttling
      uploadThroughput: -1, // No throttling
      latency: 0,
    });

    console.log('Network latency reset to normal');
  } catch (error) {
    console.error('Failed to reset network latency:', error);
  } finally {
    await client.detach();
  }
}
