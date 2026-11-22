"""
validate_dataset.py -- ResolveFlow AI synthetic dataset validator.

Usage:
    python validate_dataset.py [path/to/tickets_synthetic.csv]
                               [--groups path/to/ticket_scenario_groups.csv]

Exits with status 1 if any critical errors are found.
This script never modifies the dataset.
"""

import csv
import re
import sys
from collections import Counter, defaultdict
from pathlib import Path

REQUIRED_COLUMNS = {"ticket_id", "subject", "description", "category", "priority", "sentiment"}
ALLOWED_CATEGORIES = {"billing", "technical_support", "account", "general_inquiry"}
ALLOWED_PRIORITIES = {"low", "medium", "high", "urgent"}
ALLOWED_SENTIMENTS = {"positive", "neutral", "negative"}
MIN_EXAMPLES_PER_LABEL = 200
SHORT_TEXT_THRESHOLD = 10

# Near-duplicate similarity threshold (Jaccard on word trigrams)
NEAR_DUP_THRESHOLD = 0.70

# Subject prefixes that indicate mechanical generation
MECHANICAL_PREFIXES = ["Re:", "Follow-up:", "Issue:", "Question:", "Help needed:", "Update:", "Escalation:"]
MECHANICAL_PREFIX_WARN_PCT = 20.0  # warn if any single prefix exceeds this %

# Known sentiment-coded openers removed during Stage 9.2.4 correction.
# Source: correct_dataset.py OPENERS list — do not modify independently.
KNOWN_SENTIMENT_OPENERS = [
    # negative
    "Still dealing with an issue: ",
    "Following up on a problem — ",
    "Raising this again because ",
    "Unfortunately I need to report that ",
    "This is becoming a recurring issue — ",
    # positive
    "Wanted to share some good news — ",
    "Just a quick note to say that ",
    "Happy to report that ",
    "Pleased to let you know that ",
    "Writing to pass on some positive feedback — ",
    # neutral
    "Just a quick note — ",
    "Reaching out because ",
    "Wanted to flag that ",
    "Writing to let you know that ",
    "Checking in about something — ",
]

# Description openers that leak sentiment
SENTIMENT_OPENERS = [
    "I would like to enquire about",
    "I have a question about",
    "I am getting in touch regarding",
    "I wanted to reach out because",
    "I am writing to let you know",
    "Could you please assist me with",
    "I am contacting support because",
    "I need some help with",
    "I cannot believe that",
    "I am writing to complain about",
    "I am disappointed to report that",
    "I am extremely unhappy because",
    "I am very frustrated because",
    "I am really pleased to report that",
    "I am very happy with",
    "I have had a terrible experience with",
    "I have been dealing with a serious problem",
    "I just wanted to say that",
    "I wanted to share some positive feedback",
    "I am writing to thank you",
    "This is unacceptable",
    "I wanted to let you know",
]
OPENER_WARN_PCT = 15.0  # warn if total opener-prefixed descriptions exceed this %

# Category keyword leakage: words strongly associated with each category
CATEGORY_KEYWORDS = {
    "billing":           [r"\bbilling\b", r"\binvoice\b", r"\bpayment\b", r"\brefund\b",
                          r"\bcharge\b", r"\bsubscription\b", r"\btransaction\b"],
    "account":           [r"\baccount\b", r"\bpassword\b", r"\blogin\b", r"\bprofile\b",
                          r"\bverif\b", r"\bpermission\b"],
    "technical_support": [r"\bcrash\b", r"\bbug\b", r"\bapi\b", r"\bintegration\b",
                          r"\boutage\b", r"\bdowntime\b", r"\berror\b"],
    "general_inquiry":   [r"\benquir\b", r"\binquir\b", r"\bdemo\b", r"\bdocumentation\b",
                          r"\bpricin\b", r"\bcertif\b"],
}
LEAKAGE_WARN_PCT = 90.0  # warn if >90% of a category's records contain a strong keyword


def _stats(values):
    if not values:
        return {"min": 0, "max": 0, "mean": 0, "median": 0}
    s = sorted(values)
    n = len(s)
    return {"min": s[0], "max": s[-1], "mean": round(sum(s) / n, 1), "median": s[n // 2]}


def _bar(count, total, width=30):
    filled = round(width * count / total) if total else 0
    return "[" + "#" * filled + " " * (width - filled) + "] " + f"{count:>5}  ({100 * count / total:.1f}%)"


def _trigrams(text):
    words = re.sub(r"[^a-z0-9 ]", "", text.lower()).split()
    if len(words) < 3:
        return set(words)
    return {tuple(words[i:i+3]) for i in range(len(words) - 2)}


def _jaccard(a, b):
    if not a or not b:
        return 0.0
    inter = len(a & b)
    union = len(a | b)
    return inter / union if union else 0.0


SEP = "-" * 60


def validate(path, groups_path=None):
    print(SEP)
    print("ResolveFlow AI -- Synthetic Dataset Validation Report")
    print(SEP)
    print(f"File : {path}")

    if not path.exists():
        print("\n[CRITICAL] File not found.")
        return 1

    with open(path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        raw_columns = set(reader.fieldnames or [])
        rows = list(reader)

    print(f"Rows : {len(rows)}\n")

    critical = 0
    warnings = 0
    total = len(rows)

    # ── 1. Required columns ──────────────────────────────────────────────────
    print("-- 1. Column check " + "-" * 41)
    missing_cols = REQUIRED_COLUMNS - raw_columns
    extra_cols = raw_columns - REQUIRED_COLUMNS
    if missing_cols:
        print(f"  [CRITICAL] Missing columns: {sorted(missing_cols)}")
        critical += 1
    else:
        print("  [OK] All required columns present.")
    if extra_cols:
        print(f"  [INFO] Extra columns (ignored): {sorted(extra_cols)}")

    # ── 2. Missing values ────────────────────────────────────────────────────
    print("\n-- 2. Missing values " + "-" * 39)
    missing_counts = Counter()
    for row in rows:
        for col in REQUIRED_COLUMNS:
            if not row.get(col, "").strip():
                missing_counts[col] += 1
    if missing_counts:
        for col, n in sorted(missing_counts.items()):
            print(f"  [CRITICAL] '{col}' has {n} missing value(s).")
        critical += 1
    else:
        print("  [OK] No missing values.")

    # ── 3. Label validation ──────────────────────────────────────────────────
    print("\n-- 3. Label validation " + "-" * 37)
    bad_labels = []
    for i, row in enumerate(rows, start=2):
        if row.get("category") not in ALLOWED_CATEGORIES:
            bad_labels.append(f"Row {i}: invalid category '{row.get('category')}'")
        if row.get("priority") not in ALLOWED_PRIORITIES:
            bad_labels.append(f"Row {i}: invalid priority '{row.get('priority')}'")
        if row.get("sentiment") not in ALLOWED_SENTIMENTS:
            bad_labels.append(f"Row {i}: invalid sentiment '{row.get('sentiment')}'")
    if bad_labels:
        for msg in bad_labels[:20]:
            print(f"  [CRITICAL] {msg}")
        if len(bad_labels) > 20:
            print(f"  ... and {len(bad_labels) - 20} more.")
        critical += 1
    else:
        print("  [OK] All label values are valid.")

    # ── 4. Duplicate ticket IDs ──────────────────────────────────────────────
    print("\n-- 4. Duplicate ticket IDs " + "-" * 33)
    id_counts = Counter(row.get("ticket_id", "") for row in rows)
    dup_ids = {k: v for k, v in id_counts.items() if v > 1}
    if dup_ids:
        print(f"  [CRITICAL] {len(dup_ids)} duplicate ticket ID(s) found:")
        for tid, cnt in list(dup_ids.items())[:10]:
            print(f"    {tid}: {cnt} occurrences")
        critical += 1
    else:
        print("  [OK] All ticket IDs are unique.")

    # ── 5. Exact duplicate content ───────────────────────────────────────────
    print("\n-- 5. Exact duplicate content " + "-" * 30)
    content_keys = [(row.get("subject", ""), row.get("description", "")) for row in rows]
    content_counts = Counter(content_keys)
    exact_dups = {k: v for k, v in content_counts.items() if v > 1}
    if exact_dups:
        print(f"  [WARNING] {len(exact_dups)} exact subject+description duplicate(s).")
        for (s, _), n in list(exact_dups.items())[:5]:
            print(f"    [{n}x] \"{s[:60]}\"")
        warnings += 1
    else:
        print("  [OK] No exact content duplicates.")

    # ── 6. Conflicting labels on duplicate content ───────────────────────────
    print("\n-- 6. Conflicting labels on duplicate content " + "-" * 14)
    content_label_map = {}
    conflicts = 0
    for row in rows:
        key = (row.get("subject", ""), row.get("description", ""))
        label = (row.get("category"), row.get("priority"), row.get("sentiment"))
        if key in content_label_map:
            if content_label_map[key] != label:
                conflicts += 1
        else:
            content_label_map[key] = label
    if conflicts:
        print(f"  [CRITICAL] {conflicts} content pair(s) have conflicting labels.")
        critical += 1
    else:
        print("  [OK] No conflicting labels on duplicate content.")

    # ── 7. Minimum label counts ──────────────────────────────────────────────
    cat_counts = Counter(row.get("category") for row in rows)
    pri_counts = Counter(row.get("priority") for row in rows)
    sent_counts = Counter(row.get("sentiment") for row in rows)

    print("\n-- 7a. Category distribution " + "-" * 31)
    for label in sorted(ALLOWED_CATEGORIES):
        n = cat_counts.get(label, 0)
        flag = "  [WARNING] Below 200 examples." if n < MIN_EXAMPLES_PER_LABEL else ""
        print(f"  {label:<22} {_bar(n, total)}{flag}")
        if n < MIN_EXAMPLES_PER_LABEL:
            warnings += 1

    print("\n-- 7b. Priority distribution " + "-" * 31)
    for label in ["low", "medium", "high", "urgent"]:
        n = pri_counts.get(label, 0)
        flag = "  [WARNING] Below 200 examples." if n < MIN_EXAMPLES_PER_LABEL else ""
        print(f"  {label:<22} {_bar(n, total)}{flag}")
        if n < MIN_EXAMPLES_PER_LABEL:
            warnings += 1

    print("\n-- 7c. Sentiment distribution " + "-" * 30)
    for label in ["positive", "neutral", "negative"]:
        n = sent_counts.get(label, 0)
        flag = "  [WARNING] Below 200 examples." if n < MIN_EXAMPLES_PER_LABEL else ""
        print(f"  {label:<22} {_bar(n, total)}{flag}")
        if n < MIN_EXAMPLES_PER_LABEL:
            warnings += 1

    # ── 8. Near-duplicate detection ──────────────────────────────────────────
    print("\n-- 8. Near-duplicate detection " + "-" * 29)
    print(f"  (Jaccard trigram similarity threshold: {NEAR_DUP_THRESHOLD:.0%})")
    desc_trigrams = [_trigrams(row.get("description", "")) for row in rows]
    subj_trigrams = [_trigrams(row.get("subject", "")) for row in rows]
    near_dup_groups = []
    visited = set()
    for i in range(len(rows)):
        if i in visited:
            continue
        group = [i]
        for j in range(i + 1, len(rows)):
            if j in visited:
                continue
            if (_jaccard(desc_trigrams[i], desc_trigrams[j]) >= NEAR_DUP_THRESHOLD or
                    _jaccard(subj_trigrams[i], subj_trigrams[j]) >= NEAR_DUP_THRESHOLD):
                group.append(j)
                visited.add(j)
        if len(group) > 1:
            near_dup_groups.append(group)
            visited.add(i)
    if near_dup_groups:
        total_near_dup = sum(len(g) for g in near_dup_groups)
        print(f"  [WARNING] {len(near_dup_groups)} near-duplicate group(s) found "
              f"({total_near_dup} records involved). Review before splitting.")
        for g in near_dup_groups[:5]:
            ids = [rows[idx]["ticket_id"] for idx in g]
            cats_in_g = set(rows[idx]["category"] for idx in g)
            print(f"    Group of {len(g)}: {ids[:4]}{'...' if len(ids)>4 else ''} "
                  f"categories={cats_in_g}")
        if len(near_dup_groups) > 5:
            print(f"    ... and {len(near_dup_groups) - 5} more groups.")
        warnings += 1
    else:
        print(f"  [OK] No near-duplicate groups found at {NEAR_DUP_THRESHOLD:.0%} threshold.")

    # ── 9. Repeated subject prefixes ─────────────────────────────────────────
    print("\n-- 9. Subject prefix analysis " + "-" * 30)
    prefix_counts = Counter()
    for row in rows:
        subj = row.get("subject", "")
        matched = False
        for p in MECHANICAL_PREFIXES:
            if subj.startswith(p):
                prefix_counts[p] += 1
                matched = True
                break
        if not matched:
            prefix_counts["(none)"] += 1
    for p, n in prefix_counts.most_common():
        pct = 100 * n / total
        flag = f"  [WARNING] Single prefix exceeds {MECHANICAL_PREFIX_WARN_PCT:.0f}%." \
               if p != "(none)" and pct > MECHANICAL_PREFIX_WARN_PCT else ""
        print(f"  {p:<22} {_bar(n, total)}{flag}")
        if flag:
            warnings += 1

    # ── 10a. Known sentiment-coded opener regression check ─────────────────
    print("\n-- 10a. Known sentiment-coded opener regression check " + "-" * 8)
    print(f"  Checking {len(KNOWN_SENTIMENT_OPENERS)} known patterns from Stage 9.2.4 correction.")
    regression_hits = []  # (ticket_id, matched_pattern)
    for row in rows:
        text = row.get("subject", "") + " " + row.get("description", "")
        for op in KNOWN_SENTIMENT_OPENERS:
            # word-boundary-safe: strip trailing punctuation/space for the regex
            pat = re.escape(op.strip())
            if re.search(pat, text, re.IGNORECASE):
                regression_hits.append((row.get("ticket_id", "?"), op.strip()))
                break
    if regression_hits:
        print(f"  [CRITICAL] {len(regression_hits)} ticket(s) still contain a known "
              f"sentiment-coded opener:")
        for tid, pat in regression_hits:
            print(f"    {tid}: matched pattern '{pat}'")
        critical += 1
    else:
        print(f"  [OK] Regression check passed — no known sentiment-coded openers found.")

    # ── 10b. Description opener analysis (generic diagnostic) ────────────────
    print("\n-- 10b. Description opener analysis (generic diagnostic) " + "-" * 4)
    opener_counts = Counter()
    for row in rows:
        d = row.get("description", "")
        matched = False
        for op in SENTIMENT_OPENERS:
            if d.startswith(op):
                opener_counts[op] += 1
                matched = True
                break
        if not matched:
            opener_counts["(natural)"] += 1
    natural_n = opener_counts.get("(natural)", 0)
    templated_n = total - natural_n
    templated_pct = 100 * templated_n / total if total else 0
    print(f"  Natural openings   : {natural_n} ({100*natural_n/total:.1f}%)")
    print(f"  Templated openings : {templated_n} ({templated_pct:.1f}%)")
    if templated_pct > OPENER_WARN_PCT:
        print(f"  [WARNING] {templated_pct:.1f}% of descriptions use a mechanical opener "
              f"(threshold: {OPENER_WARN_PCT:.0f}%). Top openers:")
        for op, n in opener_counts.most_common(5):
            if op != "(natural)":
                print(f"    \"{op[:55]}\": {n}")
        warnings += 1
    else:
        print(f"  [OK] Templated opener rate {templated_pct:.1f}% is within threshold "
              f"({OPENER_WARN_PCT:.0f}%).")
    print(f"  (This is a diagnostic check only — see 10a for the regression check.)")

    # ── 11. Priority x sentiment combinations ────────────────────────────────
    print("\n-- 11. Priority x Sentiment combinations " + "-" * 19)
    pri_sent = Counter((row.get("priority"), row.get("sentiment")) for row in rows)
    header = f"  {'':10}" + "".join(f"{s:>12}" for s in ["positive", "neutral", "negative"])
    print(header)
    combo_warnings = 0
    for p in ["low", "medium", "high", "urgent"]:
        row_str = f"  {p:<10}"
        for s in ["positive", "neutral", "negative"]:
            n = pri_sent.get((p, s), 0)
            row_str += f"{n:>12}"
        print(row_str)
    # Warn on zero cells
    for p in ["low", "medium", "high", "urgent"]:
        for s in ["positive", "neutral", "negative"]:
            if pri_sent.get((p, s), 0) == 0:
                print(f"  [WARNING] Zero examples for priority={p} x sentiment={s}.")
                warnings += 1
                combo_warnings += 1
    if combo_warnings == 0:
        print("  [OK] All priority x sentiment combinations have at least one example.")

    # ── 12. Category x priority combinations ─────────────────────────────────
    print("\n-- 12. Category x Priority combinations " + "-" * 20)
    cat_pri = Counter((row.get("category"), row.get("priority")) for row in rows)
    header2 = f"  {'':24}" + "".join(f"{p:>8}" for p in ["low", "medium", "high", "urgent"])
    print(header2)
    cat_combo_warnings = 0
    for c in sorted(ALLOWED_CATEGORIES):
        row_str = f"  {c:<24}"
        for p in ["low", "medium", "high", "urgent"]:
            row_str += f"{cat_pri.get((c,p),0):>8}"
        print(row_str)
    for c in sorted(ALLOWED_CATEGORIES):
        for p in ["low", "medium", "high", "urgent"]:
            if cat_pri.get((c, p), 0) == 0:
                print(f"  [WARNING] Zero examples for category={c} x priority={p}.")
                warnings += 1
                cat_combo_warnings += 1
    if cat_combo_warnings == 0:
        print("  [OK] All category x priority combinations have at least one example.")

    # ── 13 & 14. Leakage checks ──────────────────────────────────────────────
    print("\n-- 13. Category keyword leakage " + "-" * 28)
    for cat in sorted(ALLOWED_CATEGORIES):
        patterns = CATEGORY_KEYWORDS.get(cat, [])
        cat_rows = [r for r in rows if r.get("category") == cat]
        hits = sum(
            1 for r in cat_rows
            if any(re.search(p, (r.get("subject","") + " " + r.get("description","")).lower())
                   for p in patterns)
        )
        pct = 100 * hits / len(cat_rows) if cat_rows else 0
        flag = f"  [WARNING] {pct:.0f}% contain strong category keywords (>{LEAKAGE_WARN_PCT:.0f}% threshold)." \
               if pct > LEAKAGE_WARN_PCT else ""
        print(f"  {cat:<22}: {hits}/{len(cat_rows)} ({pct:.0f}%) contain strong category keywords.{flag}")
        if flag:
            warnings += 1

    # ── 14. Sentiment opener leakage ─────────────────────────────────────────
    print("\n-- 14. Sentiment opener leakage " + "-" * 28)
    SENTIMENT_KEYWORDS = {
        "positive": [r"\bthank\b", r"\bgreat\b", r"\bexcellent\b", r"\bhappy\b",
                     r"\bpleased\b", r"\bimpressed\b", r"\bappreciat\b"],
        "negative": [r"\bfrustrat\b", r"\bdisappoint\b", r"\bunacceptable\b",
                     r"\bterrible\b", r"\bawful\b", r"\bcomplain\b", r"\bangry\b"],
    }
    SENTIMENT_LEAKAGE_WARN_PCT = 90.0
    for sent, patterns in SENTIMENT_KEYWORDS.items():
        sent_rows = [r for r in rows if r.get("sentiment") == sent]
        hits = sum(
            1 for r in sent_rows
            if any(re.search(p, (r.get("subject", "") + " " + r.get("description", "")).lower())
                   for p in patterns)
        )
        pct = 100 * hits / len(sent_rows) if sent_rows else 0
        flag = (f"  [WARNING] {pct:.0f}% contain strong sentiment keywords "
                f"(>{SENTIMENT_LEAKAGE_WARN_PCT:.0f}% threshold).")\
               if pct > SENTIMENT_LEAKAGE_WARN_PCT else ""
        print(f"  {sent:<12}: {hits}/{len(sent_rows)} ({pct:.0f}%) contain strong sentiment keywords.{flag}")
        if flag:
            warnings += 1
    # neutral: check it doesn't accidentally contain strong positive/negative words
    neutral_rows = [r for r in rows if r.get("sentiment") == "neutral"]
    all_sent_patterns = [p for ps in SENTIMENT_KEYWORDS.values() for p in ps]
    neutral_hits = sum(
        1 for r in neutral_rows
        if any(re.search(p, (r.get("subject", "") + " " + r.get("description", "")).lower())
               for p in all_sent_patterns)
    )
    neutral_pct = 100 * neutral_hits / len(neutral_rows) if neutral_rows else 0
    print(f"  {'neutral':<12}: {neutral_hits}/{len(neutral_rows)} ({neutral_pct:.0f}%) contain strong pos/neg sentiment keywords (cross-check).")

    # ── 15. Scenario group integrity ─────────────────────────────────────────
    print("\n-- 15. Scenario group integrity " + "-" * 28)
    groups_path_resolved = None
    if groups_path:
        groups_path_resolved = Path(groups_path)
    else:
        candidate = path.parent / "ticket_scenario_groups.csv"
        if candidate.exists():
            groups_path_resolved = candidate

    if groups_path_resolved and groups_path_resolved.exists():
        print(f"  Group file: {groups_path_resolved}")
        with open(groups_path_resolved, newline="", encoding="utf-8") as f:
            group_rows = list(csv.DictReader(f))
        # Check for duplicate ticket IDs within the mapping file itself
        raw_group_ids = [r["ticket_id"] for r in group_rows]
        dup_group_ids = {k: v for k, v in Counter(raw_group_ids).items() if v > 1}
        if dup_group_ids:
            print(f"  [CRITICAL] {len(dup_group_ids)} duplicate ticket ID(s) in group mapping file.")
            for tid, cnt in list(dup_group_ids.items())[:10]:
                print(f"    {tid}: {cnt} occurrences")
            critical += 1
        group_ids = {r["ticket_id"]: r["scenario_id"] for r in group_rows}
        csv_ids = {r["ticket_id"] for r in rows}
        missing_in_groups = csv_ids - set(group_ids.keys())
        extra_in_groups = set(group_ids.keys()) - csv_ids
        if missing_in_groups:
            print(f"  [WARNING] {len(missing_in_groups)} ticket ID(s) in CSV have no group mapping.")
            warnings += 1
        if extra_in_groups:
            print(f"  [WARNING] {len(extra_in_groups)} group mapping(s) reference IDs not in CSV.")
            warnings += 1
        if not missing_in_groups and not extra_in_groups and not dup_group_ids:
            scenario_counts = Counter(group_ids.values())
            sizes = Counter(scenario_counts.values())
            print(f"  [OK] All {len(csv_ids)} ticket IDs have group mappings (no duplicates).")
            print(f"  Unique scenarios : {len(scenario_counts)}")
            print(f"  Group size dist  : {dict(sorted(sizes.items()))}")
            # Cross-scenario near-dup check: the real train/test leakage risk
            cross_pairs = same_pairs = 0
            for i in range(len(rows)):
                for j in range(i + 1, len(rows)):
                    if (_jaccard(desc_trigrams[i], desc_trigrams[j]) >= NEAR_DUP_THRESHOLD or
                            _jaccard(subj_trigrams[i], subj_trigrams[j]) >= NEAR_DUP_THRESHOLD):
                        si = group_ids.get(rows[i]["ticket_id"], "?")
                        sj = group_ids.get(rows[j]["ticket_id"], "?")
                        if si == sj:
                            same_pairs += 1
                        else:
                            cross_pairs += 1
            print(f"  Near-dup pairs within same scenario  : {same_pairs}  (expected — same scenario)")
            print(f"  Near-dup pairs across scenarios      : {cross_pairs}  (train/test leakage risk)")
            if cross_pairs > 0:
                print(f"  [WARNING] {cross_pairs} cross-scenario near-duplicate pair(s) detected. "
                      f"Grouped splitting will NOT fully prevent leakage.")
                warnings += 1
            else:
                print(f"  [OK] Zero cross-scenario near-duplicate pairs. "
                      f"Grouped splitting eliminates near-dup leakage.")
    else:
        print("  [INFO] No scenario group file found. Skipping group integrity check.")
        print("         Create ticket_scenario_groups.csv for grouped train/test splitting.")

    # ── Text length statistics ────────────────────────────────────────────────
    print("\n-- 16. Text length statistics (characters) " + "-" * 17)
    subj_lens = [len(row.get("subject", "")) for row in rows]
    desc_lens = [len(row.get("description", "")) for row in rows]
    ss = _stats(subj_lens)
    ds = _stats(desc_lens)
    print(f"  Subject     min={ss['min']}  max={ss['max']}  mean={ss['mean']}  median={ss['median']}")
    print(f"  Description min={ds['min']}  max={ds['max']}  mean={ds['mean']}  median={ds['median']}")

    # ── Short text records ────────────────────────────────────────────────────
    print("\n-- 17. Short text records " + "-" * 34)
    short_subj = sum(1 for l in subj_lens if l < SHORT_TEXT_THRESHOLD)
    short_desc = sum(1 for l in desc_lens if l < SHORT_TEXT_THRESHOLD)
    if short_subj:
        print(f"  [WARNING] {short_subj} subject(s) shorter than {SHORT_TEXT_THRESHOLD} chars.")
        warnings += 1
    if short_desc:
        print(f"  [WARNING] {short_desc} description(s) shorter than {SHORT_TEXT_THRESHOLD} chars.")
        warnings += 1
    if not short_subj and not short_desc:
        print(f"  [OK] No unusually short text records (threshold: {SHORT_TEXT_THRESHOLD} chars).")

    # ── Summary ───────────────────────────────────────────────────────────────
    print("\n-- Summary " + "-" * 49)
    print(f"  Total rows      : {total}")
    print(f"  Critical errors : {critical}")
    print(f"  Warnings        : {warnings}")

    if critical:
        print("\n[RESULT] FAILED -- critical errors must be resolved before use.")
    else:
        print("\n[RESULT] PASSED -- dataset is ready for use.")

    return critical


if __name__ == "__main__":
    args = sys.argv[1:]
    groups_arg = None
    csv_arg = None
    i = 0
    while i < len(args):
        if args[i] == "--groups" and i + 1 < len(args):
            groups_arg = args[i + 1]
            i += 2
        else:
            csv_arg = args[i]
            i += 1
    csv_path = Path(csv_arg) if csv_arg else Path(__file__).parent / "tickets_synthetic.csv"
    sys.exit(validate(csv_path, groups_path=groups_arg))
