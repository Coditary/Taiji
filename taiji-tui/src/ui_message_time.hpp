#pragma once

#include <string>

enum class MessageTimeClock { Hour24, Hour12 };

/// RFC3339 UTC timestamp for the current instant (e.g. 2026-03-20T12:34:56Z).
std::string message_time_now_iso_utc();

MessageTimeClock message_time_clock_from_pref(const std::string& pref);

/// Human label for a message time: HH:MM today, date when older (German-style copy).
std::string format_message_time_label(const std::string& iso_utc, MessageTimeClock clock = MessageTimeClock::Hour24);
std::string format_message_time_label_now(MessageTimeClock clock = MessageTimeClock::Hour24);
