# Policy model

This document describes how SailGuard policies are expressed and how an endpoint receives its **effective policy**.

## Concepts

| Concept | Meaning |
|---------|---------|
| **Policy** | Named set of rules with a mode: `audit` (report only) or `enforce` (report and block) |
| **Rule** | Single match condition with action `block` or `allow` |
| **Assignment** | Binding of a policy to `organization`, `group`, or `device` |
| **Category** | Curated business label (for example messaging or P2P) expanded into technical signatures |
| **Override** | Device-specific exception, typically time-bounded for emergency allow |

## Match dimensions

Rules may match on:

- Process / executable name  
- Filesystem path (exact or prefix/contains operators as supported)  
- File content hash (SHA-256)  
- Code-signing publisher  
- Product name metadata  
- Category (expanded server-side before agent sync)

## Scope and precedence

Effective policy is always calculated for **one device**. An allow assigned to the IT group does not automatically allow every Finance device.

Precedence (highest wins when rules target the same match):

1. Device override  
2. Device-assigned policy  
3. Group-assigned policy  
4. Organization-assigned policy  

When two assignments share the same scope weight and conflict on the same match, **allow takes precedence over block** (fail-open at equal scope). Organizations that require stricter conflict behavior can revisit this rule in a future revision; the evaluation engine is centralized so behavior stays consistent across agents.

## Publish model

Administrators edit policies in draft form and **publish** to produce a new effective version for affected devices. Agents synchronize by version/hash and replace their local cache atomically.

## Agent evaluation order

For each observed process:

1. If an **allow** rule matches (and is not expired), do not block  
2. If a **block** rule matches:  
   - `audit` → record a violation event  
   - `enforce` → terminate the process and record a blocked event  
3. Critical operating-system processes are protected by a safety rail and are never terminated by policy

## Privacy posture

SailGuard focuses on application identity attributes required for policy (name, path, hash, publisher, product, account context as configured). Collection of full command lines or screen contents is not part of the default product posture.
