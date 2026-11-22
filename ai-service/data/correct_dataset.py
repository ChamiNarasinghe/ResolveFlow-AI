"""
correct_dataset.py  —  Stage 9.2.4 corrections
Applies in-place to tickets_synthetic.csv and ticket_scenario_groups.csv.
Never touches the original backup or the stage923 backup.
Run from ai-service/data/.
"""
import csv
import re
from pathlib import Path
from collections import defaultdict

CSV_PATH   = Path("tickets_synthetic.csv")
GROUP_PATH = Path("ticket_scenario_groups.csv")

# ── Load ──────────────────────────────────────────────────────────────────────
with open(CSV_PATH, encoding="utf-8") as f:
    rows = list(csv.DictReader(f))

with open(GROUP_PATH, encoding="utf-8") as f:
    group_map = {r["ticket_id"]: r["scenario_id"] for r in csv.DictReader(f)}

# ── 1. Sentiment-coded opener removal ────────────────────────────────────────
# Each opener is followed by the original description with its first char
# lowercased. We strip the opener and restore the original capitalisation.

OPENERS = [
    # negative
    "Still dealing with an issue: ",
    "Following up on a problem \u2014 ",
    "Raising this again because ",
    "Unfortunately I need to report that ",
    "This is becoming a recurring issue \u2014 ",
    # positive
    "Wanted to share some good news \u2014 ",
    "Just a quick note to say that ",
    "Happy to report that ",
    "Pleased to let you know that ",
    "Writing to pass on some positive feedback \u2014 ",
    # neutral
    "Just a quick note \u2014 ",
    "Reaching out because ",
    "Wanted to flag that ",
    "Writing to let you know that ",
    "Checking in about something \u2014 ",
]

def strip_opener(desc):
    """Remove a sentiment-coded opener and restore sentence capitalisation."""
    for op in OPENERS:
        if desc.startswith(op):
            remainder = desc[len(op):]
            # Restore capitalisation: the expand script lowercased the first char
            if remainder and remainder[0].islower():
                remainder = remainder[0].upper() + remainder[1:]
            return remainder, op.rstrip()
    return desc, None

corrections = []  # (ticket_id, old_desc, new_desc, opener_removed)
for row in rows:
    new_desc, opener = strip_opener(row["description"])
    if opener is not None:
        corrections.append({
            "ticket_id": row["ticket_id"],
            "old_desc":  row["description"],
            "new_desc":  new_desc,
            "opener":    opener,
            "sentiment": row["sentiment"],
        })
        row["description"] = new_desc

print(f"Opener corrections applied: {len(corrections)}")

# ── 2. Scenario group corrections ────────────────────────────────────────────

# 2a. A25 + A_URG4 → merge into A25
#     Both describe formal security incident reports with unusual account access.
#     Same category (account), priority (urgent), sentiment (neutral).
#     Merging prevents cross-boundary leakage.
merged_a25_a_urg4 = 0
for tid, sid in group_map.items():
    if sid == "A_URG4":
        group_map[tid] = "A25"
        merged_a25_a_urg4 += 1
print(f"A_URG4 -> A25 merges: {merged_a25_a_urg4}")

# 2b. T28 + T30 → merge into T28
#     Both describe the same production outage incident.
#     T28 = negative/emotional reports; T30 = neutral/formal reports.
#     Different sentiments but same underlying incident — must share a group.
merged_t28_t30 = 0
for tid, sid in group_map.items():
    if sid == "T30":
        group_map[tid] = "T28"
        merged_t28_t30 += 1
print(f"T30 -> T28 merges: {merged_t28_t30}")

# 2c. B28 vs T26 — keep separate.
#     B28 = billing dispute resolved (billing/high/positive).
#     T26 = technical outage resolved (technical_support/high/positive).
#     "Impressed by how [X] was resolved" is a shared phrasing pattern but
#     the underlying incidents are genuinely different. No merge.
print("B28 vs T26: kept separate (different underlying incidents).")

# 2d. A08 split into sub-scenarios by underlying issue
#     A08 mixes 5 distinct low/negative account issues:
#       A08a — confusing profile/settings navigation (UI confusion)
#       A08b — profile photo upload failing (technical failure)
#       A08c — notification settings page not loading (technical failure)
#       A08d — account setup confusion (onboarding friction)
#       A08e — verification delay (process delay)

A08_SUBGROUPS = {
    # profile/settings navigation confusion
    "A08a": {
        "Profile settings are confusing and frustrating",
        "Can't figure out how to update my profile",
        "Unable to figure out how to update my profile",
        "Account settings are poorly organised",
    },
    # profile photo upload failing
    "A08b": {
        "Profile photo upload keeps failing",
        "Profile photo upload fails repeatedly",
    },
    # notification settings page not loading
    "A08c": {
        "Notification settings page won't load",
    },
    # account setup confusion
    "A08d": {
        "Account setup was confusing and took too long",
        "Account setup was confusing and is overdue",
    },
    # verification delay
    "A08e": {
        "Verification process is unnecessarily slow",
    },
}

# Build reverse map: subject → sub-group
subj_to_subgroup = {}
for sg, subjects in A08_SUBGROUPS.items():
    for s in subjects:
        subj_to_subgroup[s] = sg

a08_splits = defaultdict(int)
unmatched_a08 = []
for row in rows:
    tid = row["ticket_id"]
    if group_map.get(tid) == "A08":
        subj = row["subject"]
        sg = subj_to_subgroup.get(subj)
        if sg:
            group_map[tid] = sg
            a08_splits[sg] += 1
        else:
            unmatched_a08.append((tid, subj))

print(f"A08 splits: {dict(a08_splits)}")
if unmatched_a08:
    print(f"  WARNING: {len(unmatched_a08)} A08 tickets not matched to sub-group:")
    for tid, subj in unmatched_a08:
        print(f"    {tid}: {repr(subj)}")

# ── 3. Write corrected files ──────────────────────────────────────────────────
with open(CSV_PATH, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["ticket_id","subject","description",
                                       "category","priority","sentiment"])
    w.writeheader()
    w.writerows(rows)

with open(GROUP_PATH, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["ticket_id","scenario_id"])
    w.writeheader()
    for tid, sid in group_map.items():
        w.writerow({"ticket_id": tid, "scenario_id": sid})

print(f"\nWrote {len(rows)} rows to {CSV_PATH}")
print(f"Wrote {len(group_map)} group mappings to {GROUP_PATH}")

# ── 4. Print before/after examples ───────────────────────────────────────────
print("\n--- Before/after examples (first 6 corrections) ---")
for c in corrections[:6]:
    print(f"[{c['ticket_id']}] sentiment={c['sentiment']}")
    print(f"  BEFORE: {c['old_desc'][:110]}")
    print(f"  AFTER : {c['new_desc'][:110]}")
    print()
