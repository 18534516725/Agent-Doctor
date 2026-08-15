import { readFileSync } from 'node:fs';

import { describe, expect, it } from 'vitest';

const styles = readFileSync('src/styles.css', 'utf8');

describe('TaskGuardian responsive contract', () => {
  it('reflows from three columns when its content area becomes narrow', () => {
    expect(styles).toContain('.overview-main { min-width:0;display:grid;gap:22px;container-type:inline-size;');
    expect(styles).toContain('@container (max-width:760px)');
    expect(styles).toContain('.guardian-core{grid-template-columns:78px minmax(0,1fr)');
    expect(styles).toContain('.guardian-message h2{font-size:clamp(32px,7cqw,48px)');
  });
});
