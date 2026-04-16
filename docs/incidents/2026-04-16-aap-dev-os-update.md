# Incident: AAP DEV Controller Broken After OS Security Update

**Date:** 2026-04-16
**Environment:** AAP DEV — flxautomationctldev02.devops.fluxys.cloud
**Impact:** Automation Controller UI completely unavailable (500 + white screen)
**Duration:** ~30 minutes from detection to full recovery
**Root cause:** OS security update pulled AAP package bump (4.5.6 → 4.5.32) with pending DB migrations and stale metadata

---

## What happened

A security update (`dnf update --security`) was applied to the AAP DEV controller node. It had been 2 years since the last update — 266 packages were updated in one batch.

The AAP repo (`ansible-automation-platform-2.4-for-rhel-9-x86_64-rpms`) was not excluded, so the `automation-controller` package was upgraded from 4.5.6 to 4.5.32 alongside the OS packages.

---

## Symptoms observed

1. **FleetDesk probes fleet** showed ctl02 returning HTTP 200 but status DEGRADED (expired TLS cert)
2. Navigating to the AAP UI showed a white screen
3. The `/api/v2/ping/` endpoint returned HTTP 500

---

## Diagnostic sequence

### Step 1: Check service status

```
systemctl status automation-controller
```

Result: `active (exited)` with `ExecStart=/bin/true` — this is a stub service, not the real controller. The actual processes are managed by supervisord.

### Step 2: Check supervisord

```
sudo supervisorctl status
```

Result: 6 of 8 processes FATAL, only uwsgi running.

```
awx-dispatcher           FATAL   Exited too quickly
awx-callback-receiver    FATAL   Exited too quickly
awx-daphne               FATAL   Exited too quickly
awx-wsrelay              FATAL   Exited too quickly
awx-ws-heartbeat         FATAL   Exited too quickly
awx-rsyslog-configurer   FATAL   Exited too quickly
awx-uwsgi                RUNNING
awx-rsyslogd             BACKOFF
```

### Step 3: Check tower.log

```
sudo tail -100 /var/log/tower/tower.log
```

Key finding:
```
AWX is currently migrating, retry in 10s...
```

Repeated every 10 seconds. AAP was stuck in "migration mode".

### Step 4: Check pending migrations

```
sudo -u awx awx-manage showmigrations --plan | grep '\[ \]'
```

Result: **7 unapplied Django migrations**

```
[ ]  main.0188_add_bitbucket_dc_webhook
[ ]  social_django.0011_alter_id_fields
[ ]  social_django.0012_usersocialauth_extra_data_new
[ ]  social_django.0013_migrate_extra_data
[ ]  social_django.0014_remove_usersocialauth_extra_data
[ ]  social_django.0015_rename_extra_data_new_usersocialauth_extra_data
[ ]  social_django.0016_alter_usersocialauth_extra_data
```

The OS update pulled a new AAP package version that included new migrations. Without running them, AAP refuses to start most services.

### Step 5: Apply migrations

```
sudo -u awx awx-manage migrate
```

Result: All 7 migrations applied successfully. After restart, dispatcher, callback-receiver, wsrelay, ws-heartbeat, rsyslog-configurer came back RUNNING.

### Step 6: Daphne still FATAL

```
sudo tail -30 /var/log/supervisor/awx-daphne.log
```

Error:
```
Exception: Missing or incorrect metadata for controller version.
Ensure controller was installed using the setup playbook.
```

Root cause: `/var/lib/awx/.tower_version` contained `4.5.6` but the installed package was now `4.5.32`. Daphne's `asgi.py` compares these and refuses to start on mismatch.

### Step 7: Fix version file

```
rpm -q --queryformat "%{VERSION}" automation-controller > /var/lib/awx/.tower_version
chown awx:awx /var/lib/awx/.tower_version
supervisorctl restart tower-processes:awx-daphne
```

Result: Daphne RUNNING. All critical processes up.

### Step 8: UI still white screen (500)

```
sudo tail -30 /var/log/supervisor/awx-uwsgi.log
```

Error:
```
--- no python application found, check your startup logs for errors ---
```

uwsgi had been running since before the fixes — it loaded the broken state and cached it. A full restart was needed:

```
supervisorctl restart all
```

### Step 9: UI still white screen (200 but blank)

The HTML loaded (200) but JavaScript/CSS assets returned 404 — the new package version has different asset hashes. Static files needed to be re-collected:

```
chown -R awx:awx /var/lib/awx/public/static/
sudo -u awx awx-manage collectstatic --noinput
systemctl restart nginx
```

Result: 542 static files copied. UI functional after browser hard refresh.

---

## Complete fix sequence (for PRD)

On each AAP controller node, after OS security update:

```bash
# 1. Check for pending migrations
sudo -u awx awx-manage showmigrations --plan | grep '\[ \]'

# 2. Apply migrations (if any)
sudo -u awx awx-manage migrate

# 3. Sync .tower_version with installed package
rpm -q --queryformat "%{VERSION}" automation-controller > /var/lib/awx/.tower_version
chown awx:awx /var/lib/awx/.tower_version

# 4. Fix static file ownership and re-collect
chown -R awx:awx /var/lib/awx/public/static/
sudo -u awx awx-manage collectstatic --noinput

# 5. Restart all AAP services
supervisorctl restart all

# 6. Restart nginx
systemctl restart nginx

# 7. Verify
curl -sk https://localhost/api/v2/ping/ | python3 -m json.tool
supervisorctl status
```

---

## What FleetDesk detected

- Probes fleet showed ctl02 as UP (HTTP 200) even when the UI was broken (white screen returning 200 with empty body)
- The probe correctly detected ctl01 as DOWN (unreachable)
- TLS cert expiry was surfaced (expired since June 2025)

## What FleetDesk could have detected but didn't

- The 500 error phase was visible in probes (expected 200, got 500 → DOWN) but only briefly
- No visibility into supervisord process health from the probes view
- No visibility into pending migrations or version mismatches
- No way to run the fix sequence from FleetDesk
- No pre/post update health check workflow

---

## Nodes affected

| Node | Role | Status after update |
|------|------|-------------------|
| flxautomationctldev01 | Controller | DOWN (unreachable — separate issue, pre-existing) |
| flxautomationctldev02 | Controller | Fixed with above sequence |
| flxautomationhubdev01 | Hub | Not yet checked — same update was applied, may need same treatment |

---

## PRD considerations

- PRD has the same AAP repo enabled — OS security updates WILL pull AAP package bumps
- The fix sequence must be applied to BOTH controllers (HA pair) and the hub
- Order matters: migrate on one controller first, verify, then the second
- PRD has stricter change management — this should be automated, not ad-hoc SSH
