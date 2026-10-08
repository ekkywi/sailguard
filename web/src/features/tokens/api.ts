import { apiFetch } from "../../lib/api"
import type {
    CreateEnrollmentTokenInput,
    EnrollmentToken,
    EnrollmentTokenListData,
} from "./types"

export function listEnrollmentTokens() {
    return apiFetch<EnrollmentTokenListData>('/v1/enrollment-tokens')
}

export function createEnrollmentToken(body: CreateEnrollmentTokenInput) {
    return apiFetch<EnrollmentToken>('/v1/enrollment-tokens', {
        method: 'POST',
        body: JSON.stringify(body),
    })
}

export function revokeEnrollmentToken(id: string) {
    return apiFetch<EnrollmentToken>(`/v1/enrollment-tokens/${id}/revoke`, {
      method: 'POST',
    })
  }