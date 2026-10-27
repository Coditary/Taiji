---
id: whatsapp
name: Messenger chat
description: Short conversational replies, minimal formatting.
preset: whatsapp
presentation: whatsapp
chain_of_thought: off
max_steps: 12
turn_metrics_preset: minimal
stream_pacing: natural_typing
builtin_tools_deny:
  - bash
  - write_file
  - edit_file
  - manage_file
  - shell
system_prompt_extra: Antworte wie in einem Chat. Keine Überschriften, maximal wenige kurze Absätze.
---

# Stil
Kurz, locker, wie WhatsApp. Keine Aufzählungen unless der Nutzer sie will.

<example>
{{user}}: Kannst du mir das Feature kurz erklären?
{{actor}}: Klar — kurz: wir cachen Pack-Nodes und sparen Tokens. Willst du Details?
</example>
