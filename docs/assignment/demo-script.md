# Demo script (5–10 minutes)

**Live app:** https://gitactionflow.vercel.app  
**Demo repo:** `Revati-Firke/Daily-Impression-AI-Model`  
**Slack channel:** `#gitactionflow-demo` (or your configured channel)

### Before you start

1. Wake backend: open https://gitactionflow-backend.onrender.com/health  
2. Rules use labels that **already exist** on the repo (or only Slack/comment rules).  
3. Disable rules that target missing labels (avoids event **Failed** while Slack succeeds).

### Suggested demo rule

```text
When an issue contains "bug"
→ Slack notification: "Demo: bug issue matched"
→ (optional) GitHub label: use an existing label name
→ (optional) GitHub comment: "Triaged by GitActionFlow"
```

---

## Walkthrough

1. **Open** https://gitactionflow.vercel.app → Login with GitHub.  
2. **Show** connected repository card.  
3. **Show** Automation rules (create/edit/enable).  
4. On GitHub, **create an issue** titled e.g. `demo bug for reviewer`.  
5. Return to dashboard → **Refresh** → show **Recent events** (processed).  
6. Open **Actions** → show Slack / label / comment statuses.  
7. Open Slack → show the notification (no secrets in the text).  
8. On GitHub, confirm label/comment if those rules ran.  
9. (Optional) Open a small **PR** with `bug` in title/body → show PR event.  
10. (Optional) GitHub webhook **Redeliver** → explain no duplicate logical action.  
11. **Architecture (30s):** signed webhook → Postgres event → worker → rules → GitHub/Slack → dashboard.  
12. **Security (30s):** HMAC signature, delivery ID idempotency, HttpOnly sessions, no secrets in frontend.  
13. **Optional AI:** available behind `AI_ENABLED`; core path works without it.  
14. **Logout** → `/dashboard` requires login again.

Keep talking points short; let the live UI prove the flow.
