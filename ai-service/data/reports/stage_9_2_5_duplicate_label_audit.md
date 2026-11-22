# Stage 9.2.5 — Duplicate Label Audit Report

**Generated:** Stage 9.2.5 completion
**Dataset:** `ai-service/data/tickets_synthetic.csv`
**Scenario mapping:** `ai-service/data/ticket_scenario_groups.csv`
**Total records in dataset:** 1,597
**Validator:** Read-only audit — no records modified

---

## Methodology

Exact duplicates are identified by normalising (strip leading/trailing whitespace) the
`subject` and `description` fields and grouping records with identical combined values.
Label consistency is then checked across `category`, `priority`, and `sentiment` for
every group. Scenario membership is checked against `ticket_scenario_groups.csv`.

---

## Summary

| Metric | Count |
|---|---|
| Total exact duplicate groups | **146** |
| Total affected records | **389** |
| Groups with identical category, priority **and** sentiment | **146** |
| Groups with conflicting category labels | **0** |
| Groups with conflicting priority labels | **0** |
| Groups with conflicting sentiment labels | **0** |
| Groups spanning multiple scenario IDs | **0** |

---

## Findings

### 1. Total exact duplicate groups: 146

There are 146 groups of records that share an identical normalised
`subject + description` pair.

**Group size distribution:**

| Records per group | Number of groups |
|---|---|
| 2 | 49 |
| 3 | 97 |

All 146 groups contain either 2 or 3 records. No group exceeds 3 records.

### 2. Total affected records: 389

389 out of 1,597 records (24.4%) participate in at least one exact duplicate group.
The remaining 1,208 records have unique content.

### 3. Groups with identical category, priority and sentiment: 146

**All 146 groups are fully label-consistent.** Every record within a duplicate group
carries the same `category`, `priority`, and `sentiment` values as all other members
of that group. There are zero label disagreements across the entire dataset.

Representative sample (5 of 146 groups):

| Subject (truncated) | IDs | category | priority | sentiment | scenario |
|---|---|---|---|---|---|
| Old address still appearing on invoices | TKT-0199, TKT-0207 | billing | medium | neutral | B18 |
| Long-range report generation fails every time | TKT-0628, TKT-0627, TKT-0617 | technical_support | medium | neutral | T13 |
| Receipt missing from my inbox | TKT-0016, TKT-0036, TKT-0020 | billing | low | neutral | B05 |
| Platform down all morning — no communication | TKT-0807, TKT-0788, TKT-0785 | technical_support | high | negative | T25 |
| Enterprise licensing options for a 200-person company | TKT-1465, TKT-1452, TKT-1441 | general_inquiry | medium | neutral | G13 |

### 4. Groups with conflicting category labels: 0

No duplicate group contains records with differing `category` values.

### 5. Groups with conflicting priority labels: 0

No duplicate group contains records with differing `priority` values.

### 6. Groups with conflicting sentiment labels: 0

No duplicate group contains records with differing `sentiment` values.

### 7. Groups spanning multiple scenario IDs: 0

Every duplicate group maps entirely to a single `scenario_id`. No duplicate pair or
triple crosses a scenario boundary. This means exact-duplicate content cannot cause
train/test leakage when grouped splitting is applied.

---

## Interpretation

The duplicate records are intentional by design: the dataset expansion strategy
(Stage 9.2.3) generates multiple surface-form variants of the same underlying scenario
ticket. Each variant differs only in appended trailing phrases (e.g.
`"Happy to provide more information if needed."`,
`"I'd appreciate a response when you get a chance."`) while preserving the core
subject and description text.

Because all variants within a group share the same labels **and** the same scenario ID,
these duplicates:

- Do **not** introduce label noise.
- Do **not** create cross-scenario leakage risk.
- Are **expected** artefacts of the expansion process, not errors.

---

## Recommendation

No corrective action is required. The duplicate groups are consistent and
scenario-contained. They should be preserved as-is to maintain dataset size and
scenario balance.

If future work requires deduplication (e.g. for a held-out evaluation set), records
should be deduplicated **within** scenario groups, not across them, to avoid
inadvertently removing all variants of a scenario from the dataset.

---

## Files examined (read-only, not modified)

- `ai-service/data/tickets_synthetic.csv` — 1,597 records, unchanged
- `ai-service/data/ticket_scenario_groups.csv` — 1,597 mappings, unchanged
