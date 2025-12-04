## Ålandsbanken

Ålandsbanken uses a single HTML page containing both list rates and average rates. It's a Finnish bank operating in Sweden.

### Fetch Page

**Minimal working request:**

```bash
curl -s 'https://www.alandsbanken.se/banktjanster/lana-pengar/bolan' \
  -H 'User-Agent: Mozilla/5.0'
```

### List Rates Table

**Table identifier:** Search for text "Aktuella räntor:" before the table

**Table structure:**
| Bindningstid | Räntesats % | Senaste ränteförändring | Förändring % |
|--------------|-------------|-------------------------|--------------|
| 3 mån | 3,85 % | 2025.10.03 | - 0,15 % |
| 1 år | 3,45 % | 2025.10.03 | - 0,15 % |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år)
- Rate: Swedish decimal format with comma (3,85 %)
- Date: YYYY.MM.DD format (e.g., 2025.10.03)

### Average Rates Table

**Table identifier:** Search for text "Genomsnittlig bolåneränta" before the table

**Table structure:**
| Bindningstid | Genomsnittlig bolåneränta | Månad |
|--------------|---------------------------|-------|
| 3 mån | 2,59 % | Oktober 2025 |
| 3 mån | 2,82 % | September 2025 |

**Data formats:**

- Term: Only 3 mån available (Ålandsbanken only publishes average rates for 3 month term)
- Rate: Swedish decimal format with comma
- Month: Swedish month name + year (e.g., "Oktober 2025")

**Note:** Unlike other banks, Ålandsbanken only provides average rate data for the 3 month term.

---

