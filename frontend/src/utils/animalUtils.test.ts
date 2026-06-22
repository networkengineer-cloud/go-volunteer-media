import { describe, it, expect } from 'vitest';
import { formatAnimalStatus } from './animalUtils';

describe('formatAnimalStatus', () => {
  it('formats bite_quarantine as the special-case label', () => {
    expect(formatAnimalStatus('bite_quarantine')).toBe('Bite Quarantine');
  });

  it('title-cases a single-word status', () => {
    expect(formatAnimalStatus('available')).toBe('Available');
  });

  it('replaces all underscores and title-cases each word', () => {
    expect(formatAnimalStatus('long_term_foster')).toBe('Long Term Foster');
  });

  it('formats under_vet_care as Under Vet Care', () => {
    expect(formatAnimalStatus('under_vet_care')).toBe('Under Vet Care');
  });

  it('returns an already-uppercase string unchanged', () => {
    expect(formatAnimalStatus('ARCHIVED')).toBe('ARCHIVED');
  });
});
