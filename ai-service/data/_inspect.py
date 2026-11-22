import csv
from collections import Counter

with open("tickets_synthetic.csv", encoding="utf-8") as f:
    rows = list(csv.DictReader(f))
with open("ticket_scenario_groups.csv", encoding="utf-8") as f:
    grp_rows = list(csv.DictReader(f))
groups = {r["ticket_id"]: r["scenario_id"] for r in grp_rows}
total = len(rows)

print("TOTAL ROWS:", total)
print("COLUMNS:", list(rows[0].keys()))

required = ["ticket_id", "subject", "description", "category", "priority", "sentiment"]
print("\nMISSING VALUES:")
for col in required:
    m = sum(1 for r in rows if not r.get(col, "").strip())
    print(f"  {col}: {m}")

id_counts = Counter(r["ticket_id"] for r in rows)
dups = {k: v for k, v in id_counts.items() if v > 1}
print(f"\nDUPLICATE TICKET IDs: {len(dups)}")
for k, v in list(dups.items())[:5]:
    print(f"  {k}: {v}x")

cats  = Counter(r["category"]  for r in rows)
pris  = Counter(r["priority"]  for r in rows)
sents = Counter(r["sentiment"] for r in rows)

print("\nCATEGORY DISTRIBUTION:")
for k in sorted(cats):
    print(f"  {k:<25}: {cats[k]:4d}  ({100*cats[k]/total:.1f}%)")

print("\nPRIORITY DISTRIBUTION:")
for k in ["low", "medium", "high", "urgent"]:
    print(f"  {k:<10}: {pris[k]:4d}  ({100*pris[k]/total:.1f}%)")

print("\nSENTIMENT DISTRIBUTION:")
for k in ["positive", "neutral", "negative"]:
    print(f"  {k:<10}: {sents[k]:4d}  ({100*sents[k]/total:.1f}%)")

print("\nPRIORITY x SENTIMENT:")
ps = Counter((r["priority"], r["sentiment"]) for r in rows)
header = f"  {'':10}" + "".join(f"{s:>12}" for s in ["positive", "neutral", "negative"])
print(header)
for p in ["low", "medium", "high", "urgent"]:
    row_str = f"  {p:<10}"
    for s in ["positive", "neutral", "negative"]:
        n = ps.get((p, s), 0)
        flag = " ZERO" if n == 0 else ""
        row_str += f"{str(n)+flag:>12}"
    print(row_str)

print("\nSCENARIO GROUPS:")
print(f"  Total mappings in groups file: {len(grp_rows)}")
unique_scens = set(groups.values())
print(f"  Unique scenario IDs: {len(unique_scens)}")
scen_sizes = Counter(groups.values())
size_dist = Counter(scen_sizes.values())
print(f"  Group size distribution: {dict(sorted(size_dist.items()))}")

# Largest and smallest groups
top5 = scen_sizes.most_common(5)
bot5 = scen_sizes.most_common()[:-6:-1]
print(f"  5 largest scenarios: {top5}")
print(f"  5 smallest scenarios: {bot5}")

csv_ids = {r["ticket_id"] for r in rows}
grp_ids = set(groups.keys())
print(f"\nID ALIGNMENT:")
print(f"  Tickets in CSV:              {len(csv_ids)}")
print(f"  Entries in groups file:      {len(grp_ids)}")
print(f"  In CSV but missing in groups:{len(csv_ids - grp_ids)}")
print(f"  In groups but not in CSV:    {len(grp_ids - csv_ids)}")
dup_grp = {k: v for k, v in Counter(r["ticket_id"] for r in grp_rows).items() if v > 1}
print(f"  Duplicate IDs in groups file:{len(dup_grp)}")

# Exact content duplicates
content_counts = Counter((r["subject"], r["description"]) for r in rows)
exact_dups = {k: v for k, v in content_counts.items() if v > 1}
print(f"\nEXACT CONTENT DUPLICATES (subject+description): {len(exact_dups)}")
for (s, _), n in list(exact_dups.items())[:5]:
    print(f"  [{n}x] \"{s[:70]}\"")

# Scenario multi-category check
scen_cats = {}
for r in rows:
    tid = r["ticket_id"]
    sid = groups.get(tid)
    if sid:
        scen_cats.setdefault(sid, set()).add(r["category"])
mixed = {sid: cats for sid, cats in scen_cats.items() if len(cats) > 1}
print(f"\nSCENARIOS WITH MIXED CATEGORIES: {len(mixed)}")
for sid, cats in list(mixed.items())[:5]:
    print(f"  {sid}: {cats}")

# Scenario multi-priority check
scen_pris = {}
for r in rows:
    tid = r["ticket_id"]
    sid = groups.get(tid)
    if sid:
        scen_pris.setdefault(sid, set()).add(r["priority"])
mixed_pri = {sid: p for sid, p in scen_pris.items() if len(p) > 1}
print(f"\nSCENARIOS WITH MIXED PRIORITIES: {len(mixed_pri)}")
for sid, p in list(mixed_pri.items())[:5]:
    print(f"  {sid}: {p}")
