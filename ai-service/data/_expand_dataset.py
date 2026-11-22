"""
_expand_dataset.py
Reads the base tickets from the four source files and generates additional
natural-language variations per scenario to reach >=200 per label.
Writes the expanded tickets_synthetic.csv and ticket_scenario_groups.csv.
"""
import csv, random, sys
from pathlib import Path
from collections import Counter, defaultdict

from _tickets import TICKETS, TICKETS_BILLING_URGENT
from _tickets_tech import TICKETS_TECH
from _tickets_account import TICKETS_ACCOUNT, TICKETS_ACCOUNT_URGENT
from _tickets_general import TICKETS_GENERAL

OUT_CSV   = Path(__file__).parent / "tickets_synthetic.csv"
GROUP_CSV = Path(__file__).parent / "ticket_scenario_groups.csv"
BACKUP    = Path(__file__).parent / "tickets_synthetic_original_backup.csv"
assert BACKUP.exists(), "Backup missing — abort."

random.seed(42)

BASE = TICKETS + TICKETS_BILLING_URGENT + TICKETS_TECH + TICKETS_ACCOUNT + TICKETS_ACCOUNT_URGENT + TICKETS_GENERAL

# ---------------------------------------------------------------------------
# Variation templates: (subject_transform, description_transform)
# Each transform is a function(subject, description) -> (new_subject, new_desc)
# We apply these to generate additional distinct-sounding tickets per scenario.
# ---------------------------------------------------------------------------

def vary(subject, description, scenario_id, cat, pri, sent):
    """Return a list of (subj, desc, cat, pri, sent, sid) variations."""
    results = []

    # Variation 1: shorter, more direct description
    words = description.split()
    if len(words) > 20:
        # Take first two sentences
        sentences = description.replace('?', '?.').replace('!', '!.').split('.')
        sentences = [s.strip() for s in sentences if s.strip()]
        short_desc = '. '.join(sentences[:2]) + ('.' if not sentences[0].endswith('?') else '')
        if len(short_desc) > 40 and short_desc != description:
            results.append((subject, short_desc, cat, pri, sent, scenario_id))

    # Variation 2: rephrase subject slightly
    subj2 = subject
    replacements = [
        ("How do I", "Where do I"),
        ("Can't", "Unable to"),
        ("not working", "has stopped working"),
        ("not saving", "won't save"),
        ("not received", "hasn't arrived"),
        ("not arriving", "not being delivered"),
        ("not loading", "failing to load"),
        ("not syncing", "stopped syncing"),
        ("not reflecting", "not showing"),
        ("not updating", "not refreshing"),
        ("seems off", "looks incorrect"),
        ("needs updating", "needs to be updated"),
        ("needs clarification", "is unclear"),
        ("took too long", "is overdue"),
        ("taking too long", "is overdue"),
        ("keeps failing", "fails repeatedly"),
        ("keeps crashing", "crashes repeatedly"),
        ("keeps reverting", "reverts repeatedly"),
        ("expired immediately", "expires too quickly"),
        ("without my consent", "without authorisation"),
        ("without my knowledge", "without my authorisation"),
        ("without explanation", "with no explanation given"),
        ("at risk", "in jeopardy"),
        ("in error", "by mistake"),
    ]
    for old, new in replacements:
        if old.lower() in subj2.lower():
            subj2 = subj2.replace(old, new).replace(old.lower(), new.lower())
            break
    if subj2 != subject:
        results.append((subj2, description, cat, pri, sent, scenario_id))

    # Variation 3: add context sentence to description
    context_by_sent = {
        "negative": [
            " This has been going on for several days now.",
            " I've already tried the standard troubleshooting steps.",
            " This is affecting my ability to do my job.",
            " I've contacted support before about this with no resolution.",
            " The impact on my work is significant.",
        ],
        "neutral": [
            " I'd appreciate a response when you get a chance.",
            " Please let me know if you need any further details.",
            " Happy to provide more information if needed.",
            " I'm not in a rush but would appreciate a response.",
            " Any guidance would be helpful.",
        ],
        "positive": [
            " Keep up the great work.",
            " I'll definitely recommend the platform to colleagues.",
            " Looking forward to continuing to use the service.",
            " This kind of experience keeps me as a loyal customer.",
            " Really appreciate the effort your team puts in.",
        ],
    }
    contexts = context_by_sent.get(sent, context_by_sent["neutral"])
    ctx = random.choice(contexts)
    desc3 = description.rstrip('. ') + '.' + ctx
    results.append((subject, desc3, cat, pri, sent, scenario_id))

    # Variation 4: add a second sentence that provides context without
    # revealing sentiment through fixed opener phrases.
    # The sentence pool is shared across all sentiment labels so that
    # the first words of the description do not systematically predict sentiment.
    context_sentences = [
        " I wanted to make sure this was on your radar.",
        " Please let me know if you need any additional details.",
        " I am happy to provide more information if that would help.",
        " Any guidance on next steps would be appreciated.",
        " I have included all the relevant details above.",
        " Feel free to reach out if you need clarification.",
        " I would appreciate a response when you get a chance.",
        " I have already checked the help documentation without success.",
    ]
    ctx4 = random.choice(context_sentences)
    desc4 = description.rstrip(". ") + "." + ctx4
    if desc4 != description:
        results.append((subject, desc4, cat, pri, sent, scenario_id))

    return results


# Build expanded list
expanded = list(BASE)
for subj, desc, cat, pri, sent, sid in BASE:
    for v in vary(subj, desc, sid, cat, pri, sent):
        expanded.append(v)

# Check counts
def counts(tickets):
    cats  = Counter(t[2] for t in tickets)
    pris  = Counter(t[3] for t in tickets)
    sents = Counter(t[4] for t in tickets)
    return cats, pris, sents

cats, pris, sents = counts(expanded)
print(f"After first expansion: {len(expanded)} tickets")
print(f"  cats={dict(cats)}")
print(f"  pris={dict(pris)}")
print(f"  sents={dict(sents)}")

# Second pass: duplicate base tickets for any label still below 200
# Group base tickets by (cat, pri, sent)
by_label = defaultdict(list)
for t in BASE:
    by_label[(t[2], t[3], t[4])].append(t)

# For each label combo, if still short, add more variations
for _ in range(3):  # up to 3 more passes
    cats, pris, sents = counts(expanded)
    need_more = (
        any(v < 200 for v in cats.values()) or
        any(v < 200 for v in pris.values()) or
        any(v < 200 for v in sents.values())
    )
    if not need_more:
        break
    # Add another round of variations from base
    for subj, desc, cat, pri, sent, sid in BASE:
        for v in vary(subj, desc, sid, cat, pri, sent):
            expanded.append(v)

cats, pris, sents = counts(expanded)
print(f"After expansion passes: {len(expanded)} tickets")
print(f"  cats={dict(cats)}")
print(f"  pris={dict(pris)}")
print(f"  sents={dict(sents)}")

# Deduplicate exact (subject, description) pairs — keep first occurrence
seen_content = set()
deduped = []
for t in expanded:
    key = (t[0], t[1])
    if key not in seen_content:
        seen_content.add(key)
        deduped.append(t)
expanded = deduped
print(f"After dedup: {len(expanded)} tickets")

# Trim to ~1500 rows with stratified sampling to preserve label balance
TARGET = 1600
if len(expanded) > TARGET:
    # Group by (cat, pri, sent) and sample proportionally
    by_combo = defaultdict(list)
    for t in expanded:
        by_combo[(t[2], t[3], t[4])].append(t)
    ratio = TARGET / len(expanded)
    trimmed = []
    for combo, tickets in by_combo.items():
        keep = max(1, round(len(tickets) * ratio))
        trimmed.extend(random.sample(tickets, min(keep, len(tickets))))
    # Ensure no individual label drops below 200 after trimming
    # Check urgent specifically and top up if needed
    urgent_tickets = [t for t in trimmed if t[3] == 'urgent']
    if len(urgent_tickets) < 200:
        urgent_pool = [t for t in expanded if t[3] == 'urgent' and t not in trimmed]
        need = 200 - len(urgent_tickets)
        trimmed.extend(random.sample(urgent_pool, min(need, len(urgent_pool))))
    expanded = trimmed

# Assign IDs and shuffle
rows = []
for i, (subj, desc, cat, pri, sent, sid) in enumerate(expanded, start=1):
    rows.append(dict(ticket_id=f"TKT-{i:04d}", subject=subj,
                     description=desc, category=cat,
                     priority=pri, sentiment=sent, scenario_id=sid))
random.shuffle(rows)

# Write CSV
with open(OUT_CSV, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["ticket_id","subject","description",
                                       "category","priority","sentiment"])
    w.writeheader()
    for r in rows:
        w.writerow({k: r[k] for k in ["ticket_id","subject","description",
                                        "category","priority","sentiment"]})

with open(GROUP_CSV, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["ticket_id","scenario_id"])
    w.writeheader()
    for r in rows:
        w.writerow({"ticket_id": r["ticket_id"], "scenario_id": r["scenario_id"]})

# Final summary
cats  = Counter(r["category"]  for r in rows)
pris  = Counter(r["priority"]  for r in rows)
sents = Counter(r["sentiment"] for r in rows)
pri_sent = Counter((r["priority"], r["sentiment"]) for r in rows)
cat_pri  = Counter((r["category"], r["priority"])  for r in rows)
scens    = Counter(r["scenario_id"] for r in rows)

print(f"\nFinal rows: {len(rows)}")
print(f"Scenarios : {len(scens)}")
print(f"Group sizes: {dict(sorted(Counter(scens.values()).items()))}")
print(f"Categories: {dict(cats)}")
print(f"Priorities: {dict(pris)}")
print(f"Sentiments: {dict(sents)}")
print("\nPriority x Sentiment:")
for p in ["low","medium","high","urgent"]:
    for s in ["positive","neutral","negative"]:
        n = pri_sent.get((p,s),0)
        flag = " *** ZERO ***" if n == 0 else ""
        print(f"  {p:<8} x {s:<10}: {n}{flag}")

errors = 0
for label, counter in [("category", cats), ("priority", pris), ("sentiment", sents)]:
    for k, v in counter.items():
        if v < 200:
            print(f"  [FAIL] {label}={k}: {v} < 200")
            errors += 1
if errors:
    print(f"\nBUILD FAILED — {errors} label(s) below minimum.")
    sys.exit(1)
else:
    print("\nAll label minimums met. Build successful.")
