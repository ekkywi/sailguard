export type EnrollmentToken = {
    id: string
    label: string
    max_uses: number
    use_count: number
    expires_at: string | null
    revoked_at: string | null
    created_by: string | null
    created_at: string
    token?: string
}

export type EnrollmentTokenListData = {
    items: EnrollmentToken[]
}

export type CreateEnrollmentTokenInput = {
    label: string
    max_uses: number
    expires_in_hours?: number
}