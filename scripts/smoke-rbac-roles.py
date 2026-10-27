#!/usr/bin/env python3
"""Lab smoke: login + permission matrix for the five SailGuard roles.

Requires API at http://127.0.0.1:18080 and lab users from
seeds/development/001_admin.sql + 002_lab_roles.sql
(password: ChangeMe!SailGuard1).

Usage:
  python3 scripts/smoke-rbac-roles.py
"""

from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.request

BASE = "http://127.0.0.1:18080"
PASS = "ChangeMe!SailGuard1"

ACCOUNTS = [
    ("admin@sailguard.local", "super_admin"),
    ("secadmin@sailguard.local", "security_admin"),
    ("itoperator@sailguard.local", "it_operator"),
    ("auditor@sailguard.local", "auditor"),
    ("viewer@sailguard.local", "viewer"),
]

# path key → expect allow (2xx) vs deny (403)
EXPECT = {
    "super_admin": {
        "/v1/devices": True,
        "/v1/events?limit=1": True,
        "/v1/policies": True,
        "/v1/users": True,
        "/v1/audit-logs?limit=1": True,
        "/v1/alert-channels": True,
        "/v1/alert-deliveries?limit=1": True,
        "/v1/enrollment-tokens": True,
        "POST /v1/policies": True,
    },
    "security_admin": {
        "/v1/devices": True,
        "/v1/events?limit=1": True,
        "/v1/policies": True,
        "/v1/users": False,
        "/v1/audit-logs?limit=1": True,
        "/v1/alert-channels": True,
        "/v1/alert-deliveries?limit=1": True,
        "/v1/enrollment-tokens": False,
        "POST /v1/policies": True,
    },
    "it_operator": {
        "/v1/devices": True,
        "/v1/events?limit=1": True,
        "/v1/policies": True,
        "/v1/users": False,
        "/v1/audit-logs?limit=1": False,
        "/v1/alert-channels": False,
        "/v1/alert-deliveries?limit=1": False,
        "/v1/enrollment-tokens": True,
        "POST /v1/policies": False,
    },
    "auditor": {
        "/v1/devices": True,
        "/v1/events?limit=1": True,
        "/v1/policies": True,
        "/v1/users": True,
        "/v1/audit-logs?limit=1": True,
        "/v1/alert-channels": True,
        "/v1/alert-deliveries?limit=1": True,
        "/v1/enrollment-tokens": False,
        "POST /v1/policies": False,
    },
    "viewer": {
        "/v1/devices": True,
        "/v1/events?limit=1": True,
        "/v1/policies": False,
        "/v1/users": False,
        "/v1/audit-logs?limit=1": False,
        "/v1/alert-channels": False,
        "/v1/alert-deliveries?limit=1": False,
        "/v1/enrollment-tokens": False,
        "POST /v1/policies": False,
    },
}

NAV = {
    "super_admin": [
        "Overview",
        "Devices",
        "Groups",
        "Tokens",
        "Agent download",
        "Policies",
        "Categories",
        "Events",
        "Users",
        "Audit",
        "Settings",
    ],
    "security_admin": [
        "Overview",
        "Devices",
        "Groups",
        "Policies",
        "Categories",
        "Events",
        "Audit",
        "Settings",
    ],
    "it_operator": [
        "Overview",
        "Devices",
        "Groups",
        "Tokens",
        "Agent download",
        "Policies",
        "Categories",
        "Events",
    ],
    "auditor": [
        "Overview",
        "Devices",
        "Groups",
        "Policies",
        "Categories",
        "Events",
        "Users",
        "Audit",
        "Settings",
    ],
    "viewer": ["Overview", "Devices", "Groups", "Events"],
}


def req(method: str, path: str, token: str | None = None, body: dict | None = None):
    data = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        data = json.dumps(body).encode()
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=10) as res:
            raw = res.read().decode()
            return res.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            payload = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            payload = {"raw": raw}
        return e.code, payload


def login(email: str):
    status, data = req("POST", "/v1/auth/login", body={"email": email, "password": PASS})
    if status != 200 or not data.get("ok"):
        return None, status, data
    return data["data"]["access_token"], status, data


def main() -> int:
    print("=" * 72)
    print("SailGuard lab smoke — 5 roles")
    print("=" * 72)
    all_ok = True
    failures: list[str] = []

    for email, role in ACCOUNTS:
        print(f"\n### {role} ({email})")
        token, st, data = login(email)
        if not token:
            msg = f"LOGIN FAIL HTTP {st}: {data}"
            print(f"  {msg}")
            all_ok = False
            failures.append(f"{role}: login")
            continue
        print("  LOGIN OK")

        st, me = req("GET", "/v1/auth/me", token=token)
        if st != 200 or not me.get("ok"):
            print(f"  /me FAIL HTTP {st}: {me}")
            all_ok = False
            failures.append(f"{role}: /me")
            continue

        roles = me["data"].get("roles") or []
        if roles and isinstance(roles[0], dict):
            role_codes = [r.get("code", r) for r in roles]
        else:
            role_codes = list(roles)
        perms = me["data"].get("permissions") or []
        if perms and isinstance(perms[0], dict):
            perm_codes = sorted(p.get("code", p) for p in perms)
        else:
            perm_codes = sorted(perms)

        print(f"  roles: {role_codes}")
        print(f"  permissions ({len(perm_codes)}): {', '.join(perm_codes)}")
        if role not in role_codes:
            print(f"  FAIL expected role {role}")
            all_ok = False
            failures.append(f"{role}: missing role code")
        print(f"  expected nav: {', '.join(NAV[role])}")

        for key, allow in EXPECT[role].items():
            if key.startswith("POST "):
                method, path = "POST", key[5:]
                body = {
                    "name": f"smoke-{role}-{time.time_ns()}",
                    "mode": "audit",
                    "description": "lab smoke temp",
                }
            else:
                method, path = "GET", key
                body = None

            st, payload = req(method, path, token=token, body=body)
            ok_http = 200 <= st < 300
            if allow:
                passed = ok_http
                note = f"HTTP {st} (want 2xx)"
            else:
                passed = st in (401, 403)
                note = f"HTTP {st} (want 403)"

            mark = "PASS" if passed else "FAIL"
            extra = ""
            if not passed and isinstance(payload, dict) and payload.get("error"):
                extra = f" {payload['error']}"
            print(f"  {mark} {method} {path}: {note}{extra}")
            if not passed:
                all_ok = False
                failures.append(f"{role}: {method} {path}")

            if method == "POST" and ok_http and allow:
                pid = (payload.get("data") or {}).get("id")
                if pid and role in ("super_admin", "security_admin"):
                    dst, _ = req("DELETE", f"/v1/policies/{pid}", token=token)
                    print(f"       cleanup DELETE policy {pid}: HTTP {dst}")

    print("\n" + "=" * 72)
    print("SUMMARY:", "ALL PASS" if all_ok else "SOME FAILURES")
    for f in failures:
        print(f"  - {f}")
    print("=" * 72)
    return 0 if all_ok else 1


if __name__ == "__main__":
    sys.exit(main())
