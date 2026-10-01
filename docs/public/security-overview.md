# Security overview

SailGuard combines a privileged endpoint agent with a centralized policy control plane. This overview states product security principles suitable for operators and reviewers. Implementation checklists and backlog items are tracked separately in internal engineering documentation.

## Security objectives

- Protect the confidentiality and integrity of configuration and telemetry at rest and in transit within the customer’s infrastructure  
- Prevent unauthorized parties from enrolling devices or altering enforce posture  
- Limit blast radius if an individual device credential is compromised  
- Avoid turning the control plane into a general-purpose remote execution channel  
- Support accountable administration through role separation and auditability  

## Core controls

### Transport and authentication

- Agent and administrator traffic are expected to use **TLS** in production  
- Each enrolled device receives a **unique credential**; credentials are stored hashed on the server  
- Administrative APIs require authenticated principals and **RBAC** permissions  

### Authorization and change control

- Policy publish and emergency overrides are privileged operations  
- Sensitive configuration changes should be captured in an **administrative audit trail**  
- Enforcement rollout is designed to support **audit-first**, then staged enforce by group or device set  

### Endpoint safeguards

- Agents evaluate **structured policy rules** only (no arbitrary remote shell payloads)  
- A **critical-process safety rail** prevents termination of essential operating-system processes even if policy is misconfigured  
- Local agent state (credentials, policy cache, event queue) should be protected with OS ACLs appropriate to a privileged service  

### Data platform hardening

- PostgreSQL and Redis are intended to run on **private networks** with authentication enabled  
- Application secrets (signing keys, database URLs, SMTP credentials) must be supplied via environment or secret stores—not committed to source control  
- Alert webhooks should be configured carefully to reduce SSRF risk (HTTPS targets, timeouts, least data in payloads)  

### Privacy-minded telemetry

- Default telemetry focuses on attributes needed for application control decisions  
- Broad content capture (for example full command lines or screenshots) is outside the default product posture  

## Operational recommendations

- Keep database backups and practice restoration  
- Restrict who may hold policy-write and system-administration roles  
- Pilot enforce mode on a limited device set before organization-wide rollout  
- Align monitoring with internal HR/legal expectations (purpose, retention, access)  

## Assurance roadmap (non-exhaustive)

Capabilities such as multi-factor admin authentication, mutual TLS for agents, signed policy documents, and signed agent binaries are part of the product’s longer-term hardening direction. Specific delivery sequencing is maintained in internal planning documents.
