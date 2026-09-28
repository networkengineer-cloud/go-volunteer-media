import axios from 'axios';
import { groupsApi } from '../api/client';

export async function load(groupId: number) {
  // ruleid: raw-axios-http-call
  const direct = await axios.get(`/api/groups/${groupId}`);
  // ruleid: raw-axios-http-call
  await axios.post('/api/request-password-reset', { email: 'a@b.c' });
  // ruleid: raw-axios-http-call
  await axios({ url: '/api/x' });
  // ok: raw-axios-http-call
  const typed = await groupsApi.getById(groupId);
  try {
    return [direct, typed];
  } catch (err) {
    // ok: raw-axios-http-call
    if (axios.isCancel(err)) return null;
    // ok: raw-axios-http-call
    return axios.isAxiosError(err) ? err.message : null;
  }
}
