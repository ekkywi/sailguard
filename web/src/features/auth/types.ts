export type LoginData = {
    access_token: string
    token_type: string
    expires_in: number
    user: { id: string; email: string; name: string }
}

export type Me = {
    id: string
    email: string
    name: string
    roles: string[]
    permissions: string[]
}