import { apiFetch } from '../../lib/api';
import type { LoginData, Me } from './types';

export function login(email: string, password: string) {
    return apiFetch<LoginData>('/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
    })
}

export function fetchMe() {
    return apiFetch<Me>('/v1/auth/me');
}