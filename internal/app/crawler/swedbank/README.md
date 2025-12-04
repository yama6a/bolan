## Swedbank

Swedbank uses two separate HTML pages - one for list rates and one for historic average rates.

### List Rates

**Minimal working request:**

```bash
curl -s 'https://www.swedbank.se/privat/boende-och-bolan/bolanerantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Aktuella bolåneräntor – listpris" before the table

**Table structure:**
| Bindningstid | Ränta |
|--------------|-------|
| 3 mån | 3,05 % |
| 1 år | 3,17 % |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år)
- Rate: Swedish decimal format with comma (3,05 %)
- Date extracted from header: "senast ändrad 25 september 2025"

### Historic Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.swedbank.se/privat/boende-och-bolan/bolanerantor/historiska-genomsnittsrantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Find table by caption containing "Våra historiska genomsnittsräntor"

**Table structure (transposed - months as rows, terms as columns):**
| Bindningstid | 3 månader | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år | Banklån* |
|--------------|-----------|------|------|------|------|------|------|-------|----------|
| nov. 2025 | 2,65 | 2,84 | 2,84 | 2,90 | 2,99 | 3,07 | 3,34 | 3,30 | 4,88 |
| okt. 2025 | 2,68 | 2,92 | 2,93 | 3,00 | 3,10 | 3,19 | 3,50 | 3,57 | 4,98 |

**Data formats:**

- Month: Abbreviated Swedish month with period + year (e.g., "nov. 2025", "okt. 2025")
- Rate: Swedish decimal format with comma
- Missing data: "-" or empty
- Banklån* row is skipped (not a standard term)

**Month abbreviations:**

| Abbreviation | Full Name | Month |
|--------------|-----------|-------|
| jan. | januari | January |
| feb. | februari | February |
| mar. | mars | March |
| apr. | april | April |
| maj | maj | May |
| jun. | juni | June |
| jul. | juli | July |
| aug. | augusti | August |
| sep. | september | September |
| okt. | oktober | October |
| nov. | november | November |
| dec. | december | December |

---

