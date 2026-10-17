export type Policy = {
  id: string
  name: string
  description: string
  mode: 'audit' | 'enforce' | string
  priority: number
  version: number
  published_at: string | null
  created_at: string
  updated_at: string
}

export type PolicyListData = {
  items: Policy[]
}

export type CreatePolicyInput = {
  name: string
  description?: string
  mode?: string
  priority?: number
}

export type PolicyRule = {
  id: string
  policy_id: string
  action: string
  rule_type: string
  operator: string
  value: string
  category_code: string | null
  enabled: boolean
  created_at: string
  updated_at: string
}

export type PolicyRuleListData = {
  items: PolicyRule[]
}

export type AddPolicyRuleInput = {
  action: string
  rule_type: string
  operator?: string
  value?: string
  category_code?: string
}

export type UpdatePolicyRuleInput = AddPolicyRuleInput

export type PolicyAssignment = {
  id: string
  policy_id: string
  scope: 'org' | 'group' | 'device' | string
  group_id: string | null
  device_id: string | null
  created_at: string
}

export type PolicyAssignmentListData = {
  items: PolicyAssignment[]
}

export type AddPolicyAssignmentInput = {
  scope: string
  group_id?: string
  device_id?: string
}
