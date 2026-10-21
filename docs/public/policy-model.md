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

For each observed process (Linux `/proc` list; Windows Toolhelp list, WMI creation watch, and PID-diff/scan fallbacks; same evaluation rules):

1. If an **allow** rule matches (and is not expired), do not block  
2. If a **block** rule matches:  
   - `audit` → report a `violation_detected` event (persisted and shown in the admin Events console)  
   - `enforce` → terminate the process and record a `violation_blocked` event  
3. Critical operating-system processes are protected by a safety rail and are never terminated by policy  

Administrators can switch a policy between `audit` and `enforce` (PATCH + publish) and add, edit, or delete rules from the policy detail console. Rules may match technical fields (process name, path, hash, publisher, product) or a curated **category** (expanded server-side from category signatures before agents sync). The Categories console manages category metadata and the signature catalog used for that expansion.

Device overrides are managed on the device detail page. They apply only to that endpoint, win over assigned policies, and drop out of the effective policy automatically when `expires_at` passes (rows are retained for history; the console lists active overrides by default). Allow overrides require a future expiry and a reason.

## Privacy posture

SailGuard focuses on application identity attributes required for policy (name, path, hash, publisher, product, account context as configured). Collection of full command lines or screen contents is not part of the default product posture.
