export type NetworkPolicy = {
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

export type NetworkPolicyListData = {
  items: NetworkPolicy[]
}

export type CreateNetworkPolicyInput = {
  name: string
  description?: string
  mode?: string
  priority?: number
}

export type NetworkPolicyRule = {
  id: string
  policy_id: string
  action: string
  dest_type: string
  dest_value: string
  port: number | null
  label: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export type NetworkPolicyRuleListData = {
  items: NetworkPolicyRule[]
}

export type AddNetworkPolicyRuleInput = {
  action: string
  dest_type: string
  dest_value: string
  port?: number | null
  label?: string
}

export type UpdateNetworkPolicyRuleInput = AddNetworkPolicyRuleInput

export type NetworkPolicyAssignment = {
  id: string
  policy_id: string
  scope: 'org' | 'group' | 'device' | string
  group_id: string | null
  device_id: string | null
  created_at: string
}

export type NetworkPolicyAssignmentListData = {
  items: NetworkPolicyAssignment[]
}

export type AddNetworkPolicyAssignmentInput = {
  scope: string
  group_id?: string
  device_id?: string
}
