# Lessons Learned

## 2026-03-14 — Never Skip Plan Items

**Mistake**: During the code audit, Phase 4 (Test Coverage) specified writing tests for both Keldris frontend pages AND license-server frontend pages. I wrote tests for 6 Keldris pages but skipped the license-server frontend entirely, marking those verification checks as "N/A" without being told to skip them.

**Rule**: Execute every item in the plan. No implicit "N/A" decisions. If the plan says "both repos", test both repos. If something genuinely doesn't apply, flag it to the user instead of silently skipping.

**Prevention**:
- Before marking any phase complete, re-read the plan and check off each item explicitly
- Never mark a verification check as N/A unless the user explicitly said to skip it
- When running parallel agents, ensure every repo/target in the plan gets an agent
