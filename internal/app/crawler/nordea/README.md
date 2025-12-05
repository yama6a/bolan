## Nordea

Nordea uses two separate HTML pages - one for list rates and one for average rates.

### List Rates

**Minimal working request:**

```bash
curl -s 'https://www.nordea.se/privat/produkter/bolan/listrantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Listräntor för bolån" before the table

**Table structure:**

| Bindningstid | Ränta  | Ändring | Senast ändrad |
|--------------|--------|---------|---------------|
| 3 mån        | 3,33 % | -0,20   | 2025-10-06    |
| 1 år         | 3,44 % | -0,20   | 2025-07-10    |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 8 år)
- Rate: Swedish decimal format with comma (3,33 %)
- Date: YYYY-MM-DD

**Note:** Nordea has 7 standard terms (includes 8 år, does NOT have 7 år or 10 år). The page also shows 16 år and 18 år
which are not parsed.

### Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.nordea.se/privat/produkter/bolan/snittrantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "respektive månad" before the table

**Month format:** YYYYMM (e.g., 202511 = November 2025)

---

