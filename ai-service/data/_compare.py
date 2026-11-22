import csv
from collections import Counter

with open('tickets_synthetic.csv', encoding='utf-8') as f:
    cur = list(csv.DictReader(f))
with open('tickets_synthetic_stage923_backup.csv', encoding='utf-8') as f:
    bak = list(csv.DictReader(f))

print('ROW COUNTS:')
print('  Stage 9.2.3 backup:', len(bak))
print('  Current:           ', len(cur))

for label in ['category', 'priority', 'sentiment']:
    bc = Counter(r[label] for r in bak)
    cc = Counter(r[label] for r in cur)
    print(label.upper() + ' DIFF:')
    for k in sorted(set(bc) | set(cc)):
        diff = cc.get(k, 0) - bc.get(k, 0)
        sign = '+' if diff >= 0 else ''
        print('  ' + k + ': ' + str(bc.get(k, 0)) + ' -> ' + str(cc.get(k, 0)) + ' (' + sign + str(diff) + ')')

bak_ids = {r['ticket_id'] for r in bak}
cur_ids = {r['ticket_id'] for r in cur}
print('ID DIFF:')
print('  In backup not in current:', len(bak_ids - cur_ids))
print('  In current not in backup:', len(cur_ids - bak_ids))
removed = sorted(bak_ids - cur_ids)
added = sorted(cur_ids - bak_ids)
if removed:
    print('  Removed:', removed[:15])
if added:
    print('  Added:  ', added[:15])

OPENERS = [
    'Still dealing with an issue: ',
    'Following up on a problem \u2014 ',
    'Raising this again because ',
    'Unfortunately I need to report that ',
    'This is becoming a recurring issue \u2014 ',
    'Wanted to share some good news \u2014 ',
    'Just a quick note to say that ',
    'Happy to report that ',
    'Pleased to let you know that ',
    'Writing to pass on some positive feedback \u2014 ',
    'Just a quick note \u2014 ',
    'Reaching out because ',
    'Wanted to flag that ',
    'Writing to let you know that ',
    'Checking in about something \u2014 ',
]
bak_hits = sum(1 for r in bak if any(r['description'].startswith(op) for op in OPENERS))
cur_hits = sum(1 for r in cur if any(r['description'].startswith(op) for op in OPENERS))
print('SENTIMENT-CODED OPENERS:')
print('  Backup: ', bak_hits)
print('  Current:', cur_hits)


def exact_dups(rows):
    c = Counter((r['subject'], r['description']) for r in rows)
    return sum(1 for v in c.values() if v > 1), sum(v for v in c.values() if v > 1)


bg, bt = exact_dups(bak)
cg, ct = exact_dups(cur)
print('EXACT CONTENT DUPLICATES:')
print('  Backup:  ' + str(bg) + ' groups, ' + str(bt) + ' total records')
print('  Current: ' + str(cg) + ' groups, ' + str(ct) + ' total records')
