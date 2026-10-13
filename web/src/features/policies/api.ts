import { apiFetch } from '../../lib/api'
import type {
  AddPolicyAssignmentInput,
  AddPolicyRuleInput,
  CreatePolicyInput,
  Policy,
  PolicyAssignment,
  PolicyAssignmentListData,
  PolicyListData,
  PolicyRule,
  PolicyRuleListData,
} from './types'

export function listPolicies() {
  return apiFetch<PolicyListData>('/v1/policies')
}

export function createPolicy(body: CreatePolicyInput) {
  return apiFetch<Policy>('/v1/policies', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getPolicy(id: string) {
  return apiFetch<Policy>(`/v1/policies/${id}`)
}

export function listPolicyRules(policyId: string) {
  return apiFetch<PolicyRuleListData>(`/v1/policies/${policyId}/rules`)
}

export function addPolicyRule(policyId: string, body: AddPolicyRuleInput) {
  return apiFetch<PolicyRule>(`/v1/policies/${policyId}/rules`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function listPolicyAssignments(policyId: string) {
  return apiFetch<PolicyAssignmentListData>(
    `/v1/policies/${policyId}/assignments`,
  )
}

export function addPolicyAssignment(
  policyId: string,
  body: AddPolicyAssignmentInput,
) {
  return apiFetch<PolicyAssignment>(`/v1/policies/${policyId}/assignments`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function removePolicyAssignment(
  policyId: string,
  assignmentId: string,
) {
  return apiFetch<{ policy_id: string; assignment_id: string }>(
    `/v1/policies/${policyId}/assignments/${assignmentId}`,
    { method: 'DELETE' },
  )
}

export function publishPolicy(policyId: string) {
  return apiFetch<Policy>(`/v1/policies/${policyId}/publish`, {
    method: 'POST',
  })
}
