import { expect, test } from '@playwright/test'

const mockTree = {
  path: 'C:\\',
  name: 'C:',
  type: 'dir',
  sizeBytes: 1024 * 1024 * 12,
  modTime: new Date().toISOString(),
  childCount: 2,
  children: [
    {
      path: 'C:\\Projects',
      name: 'Projects',
      type: 'dir',
      sizeBytes: 1024 * 1024 * 8,
      modTime: new Date().toISOString(),
      childCount: 1,
      children: [
        {
          path: 'C:\\Projects\\alpha.zip',
          name: 'alpha.zip',
          type: 'file',
          sizeBytes: 1024 * 1024 * 8,
          modTime: new Date().toISOString(),
          childCount: 0
        }
      ]
    },
    {
      path: 'C:\\Windows',
      name: 'Windows',
      type: 'dir',
      sizeBytes: 1024 * 1024 * 4,
      modTime: new Date().toISOString(),
      childCount: 0
    }
  ]
}

test('user can analyze and see tree', async ({ page }) => {
  await page.route('**/api/scan**', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          node: mockTree,
          fromCache: false,
          modTime: new Date().toISOString(),
          scannedAt: new Date().toISOString()
        }
      })
    })
  })

  await page.goto('/')
  await page.getByRole('button', { name: 'Analyze' }).click()
  await expect(page.getByText('Projects')).toBeVisible()
  await expect(page.getByText('Windows')).toBeVisible()
})
