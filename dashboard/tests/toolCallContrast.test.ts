import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const cssPath = resolve(process.cwd(), 'src/styles.css');
const css = readFileSync(cssPath, 'utf8');

function token(name: string) {
  const match = css.match(new RegExp(`${name}:\\s*(#[0-9a-fA-F]{6})`));
  expect(match, `${name} must be a six-digit hex color`).not.toBeNull();
  return match![1];
}

function luminance(hex: string) {
  const channels = hex.slice(1).match(/.{2}/g)!.map((value) => Number.parseInt(value, 16) / 255);
  const [red, green, blue] = channels.map((value) => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue;
}

function contrast(foreground: string, background: string) {
  const values = [luminance(foreground), luminance(background)].sort((left, right) => right - left);
  return (values[0] + 0.05) / (values[1] + 0.05);
}

describe('tool transcript colors', () => {
  it('uses dedicated colors with AAA contrast instead of the theme-inverted ink token', () => {
    const surface = token('--tool-surface');
    const text = token('--tool-text');
    const label = token('--tool-label');

    expect(css).toMatch(/\.tool-call\s*{[^}]*background:var\(--tool-surface\)[^}]*color:var\(--tool-text\)/s);
    expect(css).toMatch(/\.tool-call strong\s*{[^}]*color:var\(--tool-label\)/s);
    expect(contrast(text, surface)).toBeGreaterThanOrEqual(7);
    expect(contrast(label, surface)).toBeGreaterThanOrEqual(7);
  });
});
