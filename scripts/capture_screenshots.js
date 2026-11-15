const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');

async function capture() {
  const outputDir = path.resolve(__dirname, '../docs/images');
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  console.log('Launching headless browser...');
  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--font-render-hinting=medium']
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 960, deviceScaleFactor: 2 });

    console.log('Navigating to http://localhost:8088 ...');
    await page.goto('http://localhost:8088', { waitUntil: 'domcontentloaded' });

    // Wait for the app to load and render data
    await page.waitForSelector('h1', { timeout: 10000 });
    await new Promise(r => setTimeout(r, 1500));

    // 1. Capture Dashboard Overview
    console.log('Capturing cascadepilot_dashboard.png...');
    await page.screenshot({
      path: path.join(outputDir, 'cascadepilot_dashboard.png'),
      fullPage: false
    });

    // 2. Capture Department Chargeback & Virtual Keys tab
    console.log('Navigating to Tenants tab...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const tenantsBtn = buttons.find(b => b.textContent && b.textContent.includes('Department Chargeback'));
      if (tenantsBtn) tenantsBtn.click();
    });
    await new Promise(r => setTimeout(r, 800));

    console.log('Capturing cascadepilot_tenants.png...');
    await page.screenshot({
      path: path.join(outputDir, 'cascadepilot_tenants.png'),
      fullPage: false
    });

    // 3. Capture Executive ROI Calculator tab
    console.log('Navigating to ROI Calculator tab...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const roiBtn = buttons.find(b => b.textContent && b.textContent.includes('Executive ROI Calculator'));
      if (roiBtn) roiBtn.click();
    });
    await new Promise(r => setTimeout(r, 800));

    console.log('Capturing cascadepilot_roi_calculator.png...');
    await page.screenshot({
      path: path.join(outputDir, 'cascadepilot_roi_calculator.png'),
      fullPage: false
    });

    // 4. Capture Side-by-Side Model Comparison modal
    console.log('Navigating back to Overview and opening Compare Side-by-Side modal...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const overviewBtn = buttons.find(b => b.textContent && b.textContent.includes('Cockpit Overview'));
      if (overviewBtn) overviewBtn.click();
    });
    await new Promise(r => setTimeout(r, 600));

    // Click "Compare Side-by-Side"
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const compareBtn = buttons.find(b => b.textContent && b.textContent.includes('Compare Side-by-Side'));
      if (compareBtn) compareBtn.click();
    });

    // Wait for the comparison to complete and modal to display
    await page.waitForSelector('.fixed.inset-0', { timeout: 10000 });
    await new Promise(r => setTimeout(r, 1000));

    console.log('Capturing cascadepilot_comparison.png...');
    await page.screenshot({
      path: path.join(outputDir, 'cascadepilot_comparison.png'),
      fullPage: false
    });

    // Close the modal
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const closeBtn = buttons.find(b => b.textContent && b.textContent.includes('Close Benchmark'));
      if (closeBtn) closeBtn.click();
    });
    await new Promise(r => setTimeout(r, 500));

    // 5. Capture Mobile Viewport (iPhone 14/15: 390x844)
    console.log('Switching to mobile viewport (390x844)...');
    await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2 });
    await page.goto('http://localhost:8088', { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('h1', { timeout: 10000 });
    await new Promise(r => setTimeout(r, 1000));

    console.log('Capturing cascadepilot_mobile.png...');
    await page.screenshot({
      path: path.join(outputDir, 'cascadepilot_mobile.png'),
      fullPage: false
    });

    console.log('All screenshots captured successfully!');
  } finally {
    await browser.close();
  }
}

capture().catch(err => {
  console.error('Screenshot capture failed:', err);
  process.exit(1);
});
