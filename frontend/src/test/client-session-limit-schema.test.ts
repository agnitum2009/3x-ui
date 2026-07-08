import { describe, expect, it } from 'vitest';
import { ClientFormSchema } from '@/schemas/client';

const baseForm = {
  email: 'client@example.test',
  subId: '',
  uuid: '',
  password: '',
  auth: '',
  flow: '',
  security: 'auto',
  reverseTag: '',
  totalGB: 0,
  delayedStart: false,
  delayedDays: 0,
  reset: 0,
  limitIp: 0,
  upSpeedLimit: 0,
  downSpeedLimit: 0,
  tgId: 0,
  group: '',
  comment: '',
  enable: true,
  inboundIds: [],
};

describe('ClientFormSchema sessionLimit', () => {
  it('accepts unlimited and bounded session limits', () => {
    expect(ClientFormSchema.safeParse({ ...baseForm, sessionLimit: 0 }).success).toBe(true);
    expect(ClientFormSchema.safeParse({ ...baseForm, sessionLimit: 10000 }).success).toBe(true);
  });

  it('rejects invalid session limits', () => {
    expect(ClientFormSchema.safeParse({ ...baseForm, sessionLimit: -1 }).success).toBe(false);
    expect(ClientFormSchema.safeParse({ ...baseForm, sessionLimit: 10001 }).success).toBe(false);
  });
});
