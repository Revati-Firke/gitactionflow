# Demo script (5–10 minutes)

**App:** https://gitactionflow.vercel.app  
**Repo:** `Revati-Firke/Daily-Impression-AI-Model`  
**Slack:** `#gitactionflow-demo`

### Before you start

1. Wake the backend: https://gitactionflow-backend.onrender.com/health  
2. Use labels that **already exist** on the repo (or Slack/comment-only rules).  
3. Disable any rule that would apply a missing label—otherwise Slack can succeed while the event shows Failed.

### Suggested rule

```text
When an issue contains "bug"
→ Slack: "Demo: bug issue matched"
→ optional label: an existing label name
→ optional comment: "Triaged by GitActionFlow"
```

---

## Walkthrough

1. Open the app → Login with GitHub.  
2. Show the connected repository.  
3. Show Automation rules (create/edit/enable).  
4. On GitHub, open an issue titled e.g. `demo bug for reviewer`.  
5. Dashboard → Refresh → Recent events (processed).  
6. Actions panel → Slack / label / comment status.  
7. Slack channel → notification (no secrets in the text).  
8. On GitHub, confirm label/comment if those rules ran.  
9. Optional: small PR with `bug` in the title → PR event.  
10. Optional: webhook Redeliver → no duplicate logical action.  
11. **~30s architecture:** signed webhook → Postgres → worker → rules → GitHub/Slack → dashboard.  
12. **~30s security:** HMAC, delivery ID, HttpOnly sessions, no secrets in the frontend.  
13. Optional AI is behind `AI_ENABLED`; core path works without it.  
14. Logout → dashboard requires login again.

Talk less; let the live UI carry the proof.

### If asked “why this design?”

Free Neon/Render/Vercel; OAuth App + one repo; Vercel proxy so cookies work in Incognito; optional AI never blocks Slack/label/comment. Details in `AI_NOTES.md`.
