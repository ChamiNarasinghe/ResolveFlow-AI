"""
_audit_925.py  --  Stage 9.2.5 independent audit.
Run from ai-service/data/.  Read-only.
"""
import csv, re
from collections import Counter, defaultdict
from pathlib import Path

CSV  = Path("tickets_synthetic.csv")
GRP  = Path("ticket_scenario_groups.csv")
BAK1 = Path("tickets_synthetic_original_backup.csv")
BAK2 = Path("tickets_synthetic_stage923_backup.csv")

with open(CSV, encoding="utf-8") as f:
    rows = list(csv.DictReader(f))
with open(GRP, encoding="utf-8") as f:
    groups = {r["ticket_id"]: r["scenario_id"] for r in csv.DictReader(f)}
with open(BAK2, encoding="utf-8") as f:
    bak_rows = list(csv.DictReader(f))

SEP = "-" * 60

# ── helpers ──────────────────────────────────────────────────────────────────
def trigrams(text):
    words = re.sub(r"[^a-z0-9 ]", "", text.lower()).split()
    if len(words) < 3:
        return set(words)
    return {tuple(words[i:i+3]) for i in range(len(words) - 2)}

def jaccard(a, b):
    if not a or not b:
        return 0.0
    i = len(a & b)
    u = len(a | b)
    return i / u if u else 0.0

# ── Section 1: 253 vs 249 discrepancy ────────────────────────────────────────
print(SEP)
print("SECTION 1: 253 vs 249 opener discrepancy investigation")
print(SEP)

# Exact openers used by correct_dataset.py (what was corrected)
CORRECTED_OPENERS = [
    "Still dealing with an issue: ",
    "Following up on a problem \u2014 ",
    "Raising this again because ",
    "Unfortunately I need to report that ",
    "This is becoming a recurring issue \u2014 ",
    "Wanted to share some good news \u2014 ",
    "Just a quick note to say that ",
    "Happy to report that ",
    "Pleased to let you know that ",
    "Writing to pass on some positive feedback \u2014 ",
    "Just a quick note \u2014 ",
    "Reaching out because ",
    "Wanted to flag that ",
    "Writing to let you know that ",
    "Checking in about something \u2014 ",
]

# Audit-reported openers (from Stage 9.2.3 report — may differ in punctuation)
AUDIT_OPENERS = [
    "Still dealing with an issue:",          # no trailing space in audit report
    "Following up on a problem \u2014",       # no trailing space
    "Raising this again because",            # no trailing space
    "Unfortunately I need to report that",   # no trailing space
    "This is becoming a recurring issue \u2014",
    "Wanted to share some good news \u2014",
    "Happy to report that",
    "Pleased to let you know that",
    "Writing to pass on some positive feedback \u2014",
    "Just a quick note \u2014",
    "Reaching out because",
    "Wanted to flag that",
    "Writing to let you know that",
    "Checking in about something \u2014",
]
# Note: audit did NOT list "Just a quick note to say that" as a separate opener

# Count in backup (pre-correction state)
def count_openers(row_list, opener_list):
    hits = []
    for r in row_list:
        d = r["description"]
        for op in opener_list:
            if d.startswith(op):
                hits.append((r["ticket_id"], op, d[:80]))
                break
    return hits

bak_corrected = count_openers(bak_rows, CORRECTED_OPENERS)
bak_audit     = count_openers(bak_rows, AUDIT_OPENERS)

print(f"Pre-correction backup rows: {len(bak_rows)}")
print(f"Hits using correct_dataset.py opener list: {len(bak_corrected)}")
print(f"Hits using audit-reported opener list:     {len(bak_audit)}")

# Find the 4-record difference
corrected_ids = {t for t, _, _ in bak_corrected}
audit_ids     = {t for t, _, _ in bak_audit}
in_corrected_not_audit = corrected_ids - audit_ids
in_audit_not_corrected = audit_ids - corrected_ids
print(f"\nIn corrected list but not audit list: {len(in_corrected_not_audit)}")
for tid in sorted(in_corrected_not_audit):
    r = next(x for x in bak_rows if x["ticket_id"] == tid)
    print(f"  {tid}: {r['description'][:80]}")
print(f"In audit list but not corrected list: {len(in_audit_not_corrected)}")

# The audit reported 253 — check if "Just a quick note to say that" accounts for the gap
jqn_hits = [r for r in bak_rows if r["description"].startswith("Just a quick note to say that")]
print(f"\n'Just a quick note to say that' hits in backup: {len(jqn_hits)}")
for r in jqn_hits:
    print(f"  {r['ticket_id']} [{r['sentiment']}]: {r['description'][:80]}")

# Check for any Unicode variants (em-dash vs hyphen, etc.)
print("\nChecking Unicode/punctuation variants in backup:")
variants = [
    ("Following up on a problem - ", "hyphen variant"),
    ("Following up on a problem -- ", "double-hyphen variant"),
    ("This is becoming a recurring issue - ", "hyphen variant"),
    ("Checking in about something - ", "hyphen variant"),
    ("Just a quick note - ", "hyphen variant"),
    ("Wanted to share some good news - ", "hyphen variant"),
    ("Writing to pass on some positive feedback - ", "hyphen variant"),
]
for v, label in variants:
    hits = [r["ticket_id"] for r in bak_rows if r["description"].startswith(v)]
    if hits:
        print(f"  '{v[:45]}' ({label}): {len(hits)} hits — {hits[:3]}")
    else:
        print(f"  '{v[:45]}' ({label}): 0 hits")

# ── Section 2: Current dataset integrity ─────────────────────────────────────
print(f"\n{SEP}")
print("SECTION 2: Current dataset integrity")
print(SEP)

total = len(rows)
print(f"Total rows: {total}")

# 2a. Required columns
REQUIRED = {"ticket_id","subject","description","category","priority","sentiment"}
cols = set(rows[0].keys()) if rows else set()
missing_cols = REQUIRED - cols
print(f"\n[Columns] Missing: {missing_cols if missing_cols else 'none'}")

# 2b. Missing values
mv = {c: sum(1 for r in rows if not r.get(c,"").strip()) for c in REQUIRED}
print(f"[Missing values] {mv}")

# 2c. Label validity
CATS  = {"billing","technical_support","account","general_inquiry"}
PRIS  = {"low","medium","high","urgent"}
SENTS = {"positive","neutral","negative"}
bad = [(r["ticket_id"], r["category"], r["priority"], r["sentiment"])
       for r in rows
       if r["category"] not in CATS or r["priority"] not in PRIS or r["sentiment"] not in SENTS]
print(f"[Invalid labels] {len(bad)} records")

# 2d. Duplicate IDs
dup_ids = {k:v for k,v in Counter(r["ticket_id"] for r in rows).items() if v > 1}
print(f"[Duplicate IDs] {len(dup_ids)}")

# 2e. Exact duplicate content
content_map = defaultdict(list)
for r in rows:
    content_map[(r["subject"], r["description"])].append(r["ticket_id"])
exact_dups = {k:v for k,v in content_map.items() if len(v) > 1}
cross_exact = {k:v for k,v in exact_dups.items()
               if len({groups[t] for t in v}) > 1}
print(f"[Exact content dups] total groups: {len(exact_dups)}, "
      f"cross-scenario: {len(cross_exact)}")
if cross_exact:
    for (s,_), tids in list(cross_exact.items())[:3]:
        scens = {groups[t] for t in tids}
        print(f"  CROSS-SCENARIO: '{s[:50]}' tids={tids} scens={scens}")

# 2f. Sentiment-coded openers — current dataset
remaining = [(r["ticket_id"], r["sentiment"], r["description"][:80])
             for r in rows
             if any(r["description"].startswith(op) for op in CORRECTED_OPENERS)]
print(f"\n[Sentiment-coded openers remaining] {len(remaining)}")
if remaining:
    for tid, sent, d in remaining[:5]:
        print(f"  {tid} [{sent}]: {d}")

# 2g. Additional opener variants to check
print("\n[Additional opener variant scan]")
extra_variants = [
    r"^I wanted to reach out because",
    r"^I am writing to let you know",
    r"^I am writing to complain",
    r"^I am really pleased",
    r"^I am very happy",
    r"^I am very frustrated",
    r"^I am extremely unhappy",
    r"^I cannot believe",
    r"^This is unacceptable",
    r"^I just wanted to say",
    r"^I wanted to share some positive",
    r"^I am writing to thank",
    r"^I have had a terrible",
    r"^I am disappointed",
]
for pat in extra_variants:
    hits = [r["ticket_id"] for r in rows if re.match(pat, r["description"], re.I)]
    if hits:
        print(f"  FOUND '{pat}': {len(hits)} — {hits[:3]}")
print("  (no additional variants found)" if not any(
    re.match(p, r["description"], re.I) for r in rows for p in extra_variants
) else "")

# 2h. Near-duplicate analysis
print(f"\n[Near-duplicate analysis] (Jaccard trigram >= 0.70)")
desc_tg = [trigrams(r["description"]) for r in rows]
subj_tg = [trigrams(r["subject"])     for r in rows]

cross_nd_pairs = []
within_nd_pairs = 0
for i in range(len(rows)):
    for j in range(i+1, len(rows)):
        if (jaccard(desc_tg[i], desc_tg[j]) >= 0.70 or
                jaccard(subj_tg[i], subj_tg[j]) >= 0.70):
            si = groups[rows[i]["ticket_id"]]
            sj = groups[rows[j]["ticket_id"]]
            if si == sj:
                within_nd_pairs += 1
            else:
                cross_nd_pairs.append((rows[i]["ticket_id"], rows[j]["ticket_id"], si, sj))

print(f"  Within-scenario near-dup pairs: {within_nd_pairs}")
print(f"  Cross-scenario near-dup pairs:  {len(cross_nd_pairs)}")
if cross_nd_pairs:
    for t1, t2, s1, s2 in cross_nd_pairs[:5]:
        print(f"  CROSS: {t1}({s1}) vs {t2}({s2})")

# 2i. Label distributions
cats  = Counter(r["category"]  for r in rows)
pris  = Counter(r["priority"]  for r in rows)
sents = Counter(r["sentiment"] for r in rows)
print(f"\n[Category distribution]")
for c in sorted(CATS):
    print(f"  {c:<25}: {cats[c]:4d}  ({100*cats[c]/total:.1f}%)")
print(f"[Priority distribution]")
for p in ["low","medium","high","urgent"]:
    print(f"  {p:<10}: {pris[p]:4d}  ({100*pris[p]/total:.1f}%)")
print(f"[Sentiment distribution]")
for s in ["positive","neutral","negative"]:
    print(f"  {s:<10}: {sents[s]:4d}  ({100*sents[s]/total:.1f}%)")

# 2j. Priority x Sentiment
print(f"\n[Priority x Sentiment]")
ps = Counter((r["priority"],r["sentiment"]) for r in rows)
print(f"  {'':10}" + "".join(f"{s:>12}" for s in ["positive","neutral","negative"]))
for p in ["low","medium","high","urgent"]:
    row = f"  {p:<10}"
    for s in ["positive","neutral","negative"]:
        n = ps.get((p,s),0)
        flag = " *" if n == 0 else ""
        row += f"{str(n)+flag:>12}"
    print(row)

# 2k. Category x Priority
print(f"\n[Category x Priority]")
cp = Counter((r["category"],r["priority"]) for r in rows)
print(f"  {'':26}" + "".join(f"{p:>8}" for p in ["low","medium","high","urgent"]))
for c in sorted(CATS):
    row = f"  {c:<26}"
    for p in ["low","medium","high","urgent"]:
        row += f"{cp.get((c,p),0):>8}"
    print(row)

# 2l. Scenario group integrity
scen_counts = Counter(groups.values())
print(f"\n[Scenario groups]")
print(f"  Unique scenarios: {len(scen_counts)}")
print(f"  Group size distribution: {dict(sorted(Counter(scen_counts.values()).items()))}")
missing_in_groups = {r["ticket_id"] for r in rows} - set(groups.keys())
extra_in_groups   = set(groups.keys()) - {r["ticket_id"] for r in rows}
print(f"  Tickets missing from group file: {len(missing_in_groups)}")
print(f"  Group entries with no CSV match: {len(extra_in_groups)}")

# 2m. Category keyword leakage
print(f"\n[Category keyword leakage]")
CAT_KW = {
    "billing":           [r"\bbilling\b",r"\binvoice\b",r"\bpayment\b",r"\brefund\b",r"\bcharge\b",r"\bsubscription\b"],
    "account":           [r"\baccount\b",r"\bpassword\b",r"\blogin\b",r"\bprofile\b",r"\bverif\b",r"\bpermission\b"],
    "technical_support": [r"\bcrash\b",r"\bbug\b",r"\bapi\b",r"\boutage\b",r"\bdowntime\b",r"\berror\b"],
    "general_inquiry":   [r"\benquir\b",r"\binquir\b",r"\bdemo\b",r"\bdocumentation\b",r"\bpricin\b",r"\bcertif\b"],
}
for cat, patterns in CAT_KW.items():
    cat_rows = [r for r in rows if r["category"] == cat]
    hits = sum(1 for r in cat_rows if any(
        re.search(p, (r["subject"]+" "+r["description"]).lower()) for p in patterns))
    pct = 100*hits/len(cat_rows) if cat_rows else 0
    flag = " [HIGH]" if pct > 90 else ""
    print(f"  {cat:<25}: {hits}/{len(cat_rows)} ({pct:.0f}%){flag}")

# ── Section 3: Grouped split feasibility ─────────────────────────────────────
print(f"\n{SEP}")
print("SECTION 3: Grouped split feasibility (70/15/15)")
print(SEP)

target = {"train": round(total*0.70), "val": round(total*0.15), "test": round(total*0.15)}
print(f"Total: {total}  Target: {target}")

# Greedy largest-first assignment
scen_list = sorted(scen_counts.items(), key=lambda x: -x[1])
assign = {}
sizes  = {"train":0,"val":0,"test":0}
for sid, sz in scen_list:
    best = min(sizes, key=lambda s: sizes[s]/target[s])
    assign[sid] = best
    sizes[best] += sz

print(f"Simulated sizes: {sizes}")
print(f"Percentages: { {s: round(100*n/total,1) for s,n in sizes.items()} }")

for split in ["train","val","test"]:
    split_rows = [r for r in rows if assign.get(groups[r["ticket_id"]]) == split]
    sc = Counter(r["category"]  for r in split_rows)
    sp = Counter(r["priority"]  for r in split_rows)
    ss = Counter(r["sentiment"] for r in split_rows)
    up = sum(1 for r in split_rows if r["priority"]=="urgent" and r["sentiment"]=="positive")
    missing = []
    for c in CATS:
        if sc.get(c,0)==0: missing.append(f"cat:{c}")
    for p in PRIS:
        if sp.get(p,0)==0: missing.append(f"pri:{p}")
    for s in SENTS:
        if ss.get(s,0)==0: missing.append(f"sent:{s}")
    status = "WARN" if missing else "OK"
    print(f"  {split:<6} ({len(split_rows):4d} rows) [{status}] urgent-pos={up}  "
          f"{'MISSING: '+', '.join(missing) if missing else 'all labels present'}")

# Urgent-positive detail
urg_pos_tids  = [r["ticket_id"] for r in rows if r["priority"]=="urgent" and r["sentiment"]=="positive"]
urg_pos_scens = sorted({groups[t] for t in urg_pos_tids})
print(f"\n  Urgent-positive: {len(urg_pos_tids)} tickets in {len(urg_pos_scens)} scenario(s): {urg_pos_scens}")
for sid in urg_pos_scens:
    sz = scen_counts[sid]
    sp = assign.get(sid,"?")
    print(f"    {sid}: {sz} tickets -> assigned to '{sp}'")
print(f"\n  Limitation: urgent-positive tickets are confined to {len(urg_pos_scens)} scenario(s).")
print(f"  Any split that does not receive these scenarios will have 0 urgent-positive examples.")
print(f"  With only {len(urg_pos_tids)} total urgent-positive tickets, this combination")
print(f"  cannot be guaranteed in all three splits without breaking scenario boundaries.")

# ── Section 4: File inventory ─────────────────────────────────────────────────
print(f"\n{SEP}")
print("SECTION 4: Data directory file inventory")
print(SEP)

data_dir = Path(".")
files = sorted(data_dir.glob("*"))
CLASSIFICATIONS = {
    "tickets_synthetic.csv":                    "REQUIRED — primary dataset",
    "ticket_scenario_groups.csv":               "REQUIRED — scenario group mapping",
    "validate_dataset.py":                      "REQUIRED — dataset validator",
    "_build_dataset.py":                        "REQUIRED — base dataset builder",
    "_expand_dataset.py":                       "REQUIRED — dataset expander",
    "_tickets.py":                              "REQUIRED — billing/base ticket source",
    "_tickets_tech.py":                         "REQUIRED — technical_support ticket source",
    "_tickets_account.py":                      "REQUIRED — account ticket source",
    "_tickets_general.py":                      "REQUIRED — general_inquiry ticket source",
    "tickets_synthetic_original_backup.csv":    "REQUIRED — original backup (never delete)",
    "tickets_synthetic_stage923_backup.csv":    "USEFUL — stage 9.2.3 pre-correction backup",
    "ticket_scenario_groups_stage923_backup.csv": "USEFUL — stage 9.2.3 groups backup",
    "correct_dataset.py":                       "USEFUL — idempotent correction script (reproducibility)",
    "_build_out.txt":                           "TEMPORARY — build script stdout capture",
    "dup_check.txt":                            "TEMPORARY — leftover from earlier audit",
}
for f in files:
    cls = CLASSIFICATIONS.get(f.name, "UNKNOWN")
    size = f.stat().st_size if f.is_file() else 0
    print(f"  {f.name:<50} {size:>8} bytes  [{cls}]")

print("\n[Done]")
