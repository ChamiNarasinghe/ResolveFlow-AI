"""
_build_dataset.py
Assembles tickets_synthetic.csv from the four ticket source files.
Run: python _build_dataset.py
Never modifies the backup.
"""
import csv, random, sys
from pathlib import Path
from collections import Counter

from _tickets import TICKETS
from _tickets_tech import TICKETS_TECH
from _tickets_account import TICKETS_ACCOUNT
from _tickets_general import TICKETS_GENERAL

OUT_CSV   = Path(__file__).parent / "tickets_synthetic.csv"
GROUP_CSV = Path(__file__).parent / "ticket_scenario_groups.csv"
BACKUP    = Path(__file__).parent / "tickets_synthetic_original_backup.csv"

assert BACKUP.exists(), "Backup missing — abort."

ALL = TICKETS + TICKETS_TECH + TICKETS_ACCOUNT + TICKETS_GENERAL

# Assign sequential IDs then shuffle so scenario groups are not contiguous
random.seed(42)
rows = []
for i, (subj, desc, cat, pri, sent, sid) in enumerate(ALL, start=1):
    rows.append(dict(ticket_id=f"TKT-{i:04d}", subject=subj,
                     description=desc, category=cat,
                     priority=pri, sentiment=sent, scenario_id=sid))
random.shuffle(rows)

# Write main CSV (no scenario_id column — never a model feature)
with open(OUT_CSV, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["ticket_id","subject","description",
                                       "category","priority","sentiment"])
    w.writeheader()
    for r in rows:
        w.writerow({k: r[k] for k in ["ticket_id","subject","description",
                                        "category","priority","sentiment"]})

# Write scenario group metadata
with open(GROUP_CSV, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["ticket_id","scenario_id"])
    w.writeheader()
    for r in rows:
        w.writerow({"ticket_id": r["ticket_id"], "scenario_id": r["scenario_id"]})

# Summary
cats     = Counter(r["category"]  for r in rows)
pris     = Counter(r["priority"]  for r in rows)
sents    = Counter(r["sentiment"] for r in rows)
pri_sent = Counter((r["priority"], r["sentiment"]) for r in rows)
cat_pri  = Counter((r["category"], r["priority"])  for r in rows)
scens    = Counter(r["scenario_id"] for r in rows)

print(f"Rows written : {len(rows)}")
print(f"Scenarios    : {len(scens)}")
print(f"Group sizes  : {dict(sorted(Counter(scens.values()).items()))}")
print(f"\nCategories : {dict(cats)}")
print(f"Priorities : {dict(pris)}")
print(f"Sentiments : {dict(sents)}")

print("\nPriority x Sentiment:")
for p in ["low","medium","high","urgent"]:
    for s in ["positive","neutral","negative"]:
        n = pri_sent.get((p,s),0)
        flag = " *** ZERO ***" if n == 0 else ""
        print(f"  {p:<8} x {s:<10}: {n}{flag}")

print("\nCategory x Priority:")
for c in ["billing","technical_support","account","general_inquiry"]:
    for p in ["low","medium","high","urgent"]:
        print(f"  {c:<22} x {p:<8}: {cat_pri.get((c,p),0)}")

# Validation checks
errors = 0
for label, counter, minimum in [
    ("category",  cats,  200),
    ("priority",  pris,  200),
    ("sentiment", sents, 200),
]:
    for k, v in counter.items():
        if v < minimum:
            print(f"  [FAIL] {label}={k} has only {v} examples (need {minimum})")
            errors += 1

if errors:
    print(f"\nBUILD FAILED — {errors} label(s) below minimum.")
    sys.exit(1)
else:
    print("\nAll label minimums met. Build successful.")
