import { apiFetch } from '../../lib/api'
import type {
  AddNetworkPolicyAssignmentInput,
  AddNetworkPolicyRuleInput,
  CreateNetworkPolicyInput,
  NetworkPolicy,
  NetworkPolicyAssignment,
  NetworkPolicyAssignmentListData,
  NetworkPolicyListData,
  NetworkPolicyRule,
  NetworkPolicyRuleListData,
  UpdateNetworkPolicyRuleInput,
} from './types'

export function listNetworkPolicies() {
  return apiFetch<NetworkPolicyListData>('/v1/network-policies')
}

export function createNetworkPolicy(body: CreateNetworkPolicyInput) {
  return apiFetch<NetworkPolicy>('/v1/network-policies', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getNetworkPolicy(id: string) {
  return apiFetch<NetworkPolicy>(`/v1/network-policies/${id}`)
}

export function patchNetworkPolicy(id: string, body: { mode: string }) {
  return apiFetch<NetworkPolicy>(`/v1/network-policies/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function deleteNetworkPolicy(id: string) {
  return apiFetch<{ deleted?: boolean; id?: string }>(
    `/v1/network-policies/${id}`,
    { method: 'DELETE' },
  )
}

export function listNetworkPolicyRules(policyId: string) {
  return apiFetch<NetworkPolicyRuleListData>(
    `/v1/network-policies/${policyId}/rules`,
  )
}

export function addNetworkPolicyRule(
  policyId: string,
  body: AddNetworkPolicyRuleInput,
) {
  return apiFetch<NetworkPolicyRule>(
    `/v1/network-policies/${policyId}/rules`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    },
  )
}

export function updateNetworkPolicyRule(
  policyId: string,
  ruleId: string,
  body: UpdateNetworkPolicyRuleInput,
) {
  return apiFetch<NetworkPolicyRule>(
    `/v1/network-policies/${policyId}/rules/${ruleId}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    },
  )
}

export function deleteNetworkPolicyRule(policyId: string, ruleId: string) {
  return apiFetch<{ policy_id: string; rule_id: string }>(
    `/v1/network-policies/${policyId}/rules/${ruleId}`,
    { method: 'DELETE' },
  )
}

export function listNetworkPolicyAssignments(policyId: string) {
  return apiFetch<NetworkPolicyAssignmentListData>(
    `/v1/network-policies/${policyId}/assignments`,
  )
}

export function addNetworkPolicyAssignment(
  policyId: string,
  body: AddNetworkPolicyAssignmentInput,
) {
  return apiFetch<NetworkPolicyAssignment>(
    `/v1/network-policies/${policyId}/assignments`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    },
  )
}

export function removeNetworkPolicyAssignment(
  policyId: string,
  assignmentId: string,
) {
  return apiFetch<{ policy_id: string; assignment_id: string }>(
    `/v1/network-policies/${policyId}/assignments/${assignmentId}`,
    { method: 'DELETE' },
  )
}

export function publishNetworkPolicy(policyId: string) {
  return apiFetch<NetworkPolicy>(
    `/v1/network-policies/${policyId}/publish`,
    { method: 'POST' },
  )
}
