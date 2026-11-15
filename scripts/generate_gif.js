const puppeteer = require('puppeteer');
const { GIFEncoder, quantize, applyPalette } = require('gifenc');
const { PNG } = require('pngjs');
const fs = require('fs');
const path = require('path');

async function recordGif() {
  const outputPath = path.resolve(__dirname, '../docs/images/cascadepilot_demo.gif');
  console.log('Starting demo GIF recording...');

  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--font-render-hinting=medium']
  });

  try {
    const page = await browser.newPage();
    // 1080x675 viewport with deviceScaleFactor 1 for balanced quality and fast GIF rendering
    await page.setViewport({ width: 1080, height: 675, deviceScaleFactor: 1 });

    console.log('Navigating to http://localhost:8088 ...');
    await page.goto('http://localhost:8088', { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('h1', { timeout: 10000 });
    await new Promise(r => setTimeout(r, 1200));

    const frames = [];

    async function captureFrame(delay = 180) {
      const buffer = await page.screenshot({ type: 'png' });
      const png = PNG.sync.read(buffer);
      frames.push({
        data: png.data,
        width: png.width,
        height: png.height,
        delay
      });
    }

    console.log('Phase 1: Recording Cockpit Overview...');
    for (let i = 0; i < 4; i++) {
      await captureFrame(250);
      await new Promise(r => setTimeout(r, 100));
    }

    console.log('Phase 2: Scrolling to Interactive Cascade Tester...');
    await page.evaluate(() => {
      window.scrollBy({ top: 380, behavior: 'smooth' });
    });
    for (let i = 0; i < 5; i++) {
      await captureFrame(180);
      await new Promise(r => setTimeout(r, 100));
    }

    console.log('Phase 3: Selecting preset query...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const presetBtn = buttons.find(b => b.textContent && b.textContent.includes('Distributed Proof'));
      if (presetBtn) presetBtn.click();
    });
    for (let i = 0; i < 4; i++) {
      await captureFrame(220);
      await new Promise(r => setTimeout(r, 100));
    }

    console.log('Phase 4: Sending request...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const sendBtn = buttons.find(b => b.textContent && b.textContent.includes('Send Request'));
      if (sendBtn) sendBtn.click();
    });
    for (let i = 0; i < 6; i++) {
      await captureFrame(200);
      await new Promise(r => setTimeout(r, 120));
    }

    console.log('Phase 5: Clicking Compare Side-by-Side benchmark...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const compareBtn = buttons.find(b => b.textContent && b.textContent.includes('Compare Side-by-Side'));
      if (compareBtn) compareBtn.click();
    });

    // Wait for benchmark modal to render
    await page.waitForSelector('.fixed.inset-0', { timeout: 10000 });
    for (let i = 0; i < 6; i++) {
      await captureFrame(300);
      await new Promise(r => setTimeout(r, 150));
    }

    console.log('Phase 6: Closing modal and switching to ROI Calculator...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const closeBtn = buttons.find(b => b.textContent && b.textContent.includes('Close Benchmark'));
      if (closeBtn) closeBtn.click();
    });
    await new Promise(r => setTimeout(r, 300));

    await page.evaluate(() => {
      window.scrollTo({ top: 0, behavior: 'smooth' });
      const buttons = Array.from(document.querySelectorAll('button'));
      const roiBtn = buttons.find(b => b.textContent && b.textContent.includes('Executive ROI Calculator'));
      if (roiBtn) roiBtn.click();
    });
    for (let i = 0; i < 6; i++) {
      await captureFrame(300);
      await new Promise(r => setTimeout(r, 150));
    }

    console.log(`Encoding ${frames.length} frames into animated GIF...`);
    const gif = GIFEncoder();
    const { width, height } = frames[0];

    for (let i = 0; i < frames.length; i++) {
      const frame = frames[i];
      // Quantize RGBA pixels to 256 colors palette
      const palette = quantize(frame.data, 256);
      const index = applyPalette(frame.data, palette);
      gif.writeFrame(index, width, height, {
        palette,
        delay: frame.delay,
        repeat: 0 // infinite loop
      });
      process.stdout.write(`\rEncoding frame ${i + 1}/${frames.length}`);
    }
    console.log('\nFinalizing GIF...');
    gif.finish();

    fs.writeFileSync(outputPath, Buffer.from(gif.bytes()));
    const stat = fs.statSync(outputPath);
    console.log(`✓ Demo GIF created successfully at ${outputPath} (${(stat.size / 1024 / 1024).toFixed(2)} MB)`);
  } finally {
    await browser.close();
  }
}

recordGif().catch(err => {
  console.error('GIF recording failed:', err);
  process.exit(1);
});
