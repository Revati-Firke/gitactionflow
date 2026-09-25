# Production smoke test

Use after Neon + Render + Vercel are configured. Record each result as **PASS / FAIL / BLOCKED**.

Replace placeholders with your live URLs.

```text
FRONTEND = https://<your-vercel-domain>
BACKEND  = https://<your-render-domain>
```l

---

### Test 1 — Frontend

Open `FRONTEND`.

Expected: app loads; login page or session check without blank/error crash.

Status: **NOT YET RUN**

---

### Test 2 — Backend health

```bash
curl -sS "$BACKEND/health"
curl -sS "$BACKEND/ready"
```

Expected: `{"status":"ok"}` and `{"status":"ready"}`.

Status: **NOT YET RUN**

---

### Test 3 — GitHub login

1. On `FRONTEND`, click **Continue with GitHub**.
2. Authorize.
3. Land on dashboard (authenticated).

Expected flow: Vercel → GitHub → Render callback → cookie on Render → dashboard on Vercel calling Render APIs.

Status: **NOT YET RUN**

---

### Test 4 — Repository

Connect exactly one owned repository (admin). Refresh the page.

Expected: repository card persists.

Status: **NOT YET RUN**

---

### Test 5 — Rule (label)

Create rule: event `issues`, keyword `bug`, action `github_label`, label e.g. `automation` (or an existing label).

Expected: rule appears enabled in the list.

Status: **NOT YET RUN**

---

### Test 6 — GitHub issue → label

Ensure webhook points at `$BACKEND/webhooks/github`. Create/update an issue containing `bug`.

Expected: webhook accepted → event **Processed** → action **Completed** → label on GitHub → visible in dashboard.

Status: **NOT YET RUN**

---

### Test 7 — GitHub comment

Create a `github_comment` rule; trigger a matching event.

Expected: bot comment on the issue/PR; action **Completed**.

Status: **NOT YET RUN**

---

### Test 8 — Slack

With `SLACK_WEBHOOK_URL` set on Render, create a `slack_notification` rule and trigger.

Expected: Slack message arrives; action **Completed**.

Status: **NOT YET RUN**

---

### Test 9 — Duplicate webhook

In GitHub webhook Recent Deliveries, **Redeliver** a successful delivery.

Expected: `already_received` / no duplicate event row / no duplicate actions.

Status: **NOT YET RUN**

---

### Test 10 — Dashboard history

Verify events and actions lists show statuses and failures when applicable. Logout works; `/dashboard` requires login.

Status: **NOT YET RUN**
