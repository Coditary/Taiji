---
name: buy
description: Soll etwas gekauft werden? Bewertet Nutzen, Budget und Timing.
when: User fragt ob ein Kauf sinnvoll ist oder zwischen kaufen/warten/skip wählen soll
questions:
  worth_it:
    type: noul
    instructions: "Ist der Kauf insgesamt sinnvoll und lohnenswert?"
  timing:
    type: choice
    instructions: "Wann wäre der Kauf am sinnvollsten?"
    criteria:
      - sofort
      - warten
      - nicht_kaufen
  budget_fit:
    type: score
    instructions: "Wie gut passt der Preis ins Budget?"
    criteria:
      - sehr_gut
      - ok
      - zu_teuer
---

# Buy decision

Der Agent übergibt Produkt, Preis, Budget und Kontext als `text` an `decide`.
Beispiel: `decide({"profile": "buy", "text": "MacBook 2800€, altes Laptop langsam, Budget 3000€"})`
