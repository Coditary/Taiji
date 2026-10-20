#include "ui_message_time.hpp"

#include <cstdio>
#include <ctime>

namespace {

bool parse_iso_utc(const std::string& iso, std::tm& out_utc) {
    if (iso.empty()) {
        return false;
    }
    int year = 0;
    int month = 0;
    int day = 0;
    int hour = 0;
    int minute = 0;
    int second = 0;
    if (std::sscanf(iso.c_str(), "%d-%d-%dT%d:%d:%d", &year, &month, &day, &hour, &minute, &second) < 6) {
        return false;
    }
    out_utc = {};
    out_utc.tm_year = year - 1900;
    out_utc.tm_mon = month - 1;
    out_utc.tm_mday = day;
    out_utc.tm_hour = hour;
    out_utc.tm_min = minute;
    out_utc.tm_sec = second;
    return true;
}

std::time_t utc_tm_to_time(const std::tm& utc) {
#if defined(_WIN32)
    return _mkgmtime(const_cast<std::tm*>(&utc));
#else
    return timegm(const_cast<std::tm*>(&utc));
#endif
}

int calendar_day_index(const std::tm& local) {
    return local.tm_year * 400 + local.tm_yday;
}

std::string format_hh_mm(const std::tm& local, MessageTimeClock clock) {
    char buf[16];
    if (clock == MessageTimeClock::Hour12) {
        int hour = local.tm_hour % 12;
        if (hour == 0) {
            hour = 12;
        }
        const char* suffix = local.tm_hour >= 12 ? " PM" : " AM";
        std::snprintf(buf, sizeof(buf), "%d:%02d%s", hour, local.tm_min, suffix);
        return buf;
    }
    std::snprintf(buf, sizeof(buf), "%02d:%02d", local.tm_hour, local.tm_min);
    return buf;
}

const char* german_weekday_short(int wday) {
    switch (wday) {
    case 0:
        return "So";
    case 1:
        return "Mo";
    case 2:
        return "Di";
    case 3:
        return "Mi";
    case 4:
        return "Do";
    case 5:
        return "Fr";
    case 6:
        return "Sa";
    default:
        return "";
    }
}

} // namespace

MessageTimeClock message_time_clock_from_pref(const std::string& pref) {
    if (pref == "12h") {
        return MessageTimeClock::Hour12;
    }
    return MessageTimeClock::Hour24;
}

std::string message_time_now_iso_utc() {
    const std::time_t now = std::time(nullptr);
    std::tm utc{};
#if defined(_WIN32)
    gmtime_s(&utc, &now);
#else
    gmtime_r(&now, &utc);
#endif
    char buf[32];
    std::snprintf(buf, sizeof(buf), "%04d-%02d-%02dT%02d:%02d:%02dZ", utc.tm_year + 1900, utc.tm_mon + 1,
                  utc.tm_mday, utc.tm_hour, utc.tm_min, utc.tm_sec);
    return buf;
}

std::string format_message_time_label(const std::string& iso_utc, MessageTimeClock clock) {
    std::tm utc{};
    if (!parse_iso_utc(iso_utc, utc)) {
        return "";
    }

    const std::time_t stamp = utc_tm_to_time(utc);
    if (stamp <= 0) {
        return "";
    }

    std::tm local{};
#if defined(_WIN32)
    localtime_s(&local, &stamp);
#else
    localtime_r(&stamp, &local);
#endif

    const std::time_t now = std::time(nullptr);
    std::tm now_local{};
#if defined(_WIN32)
    localtime_s(&now_local, &now);
#else
    localtime_r(&now, &now_local);
#endif

    const std::string time_part = format_hh_mm(local, clock);
    const int day_delta = calendar_day_index(now_local) - calendar_day_index(local);

    if (day_delta == 0) {
        return time_part;
    }
    if (day_delta == 1) {
        return "Gestern, " + time_part;
    }
    if (day_delta > 1 && day_delta < 7) {
        return std::string(german_weekday_short(local.tm_wday)) + " " + time_part;
    }

    char date_buf[16];
    if (local.tm_year == now_local.tm_year) {
        std::snprintf(date_buf, sizeof(date_buf), "%02d.%02d.", local.tm_mday, local.tm_mon + 1);
    } else {
        std::snprintf(date_buf, sizeof(date_buf), "%02d.%02d.%02d", local.tm_mday, local.tm_mon + 1,
                      (local.tm_year + 1900) % 100);
    }
    return std::string(date_buf) + " " + time_part;
}

std::string format_message_time_label_now(MessageTimeClock clock) {
    return format_message_time_label(message_time_now_iso_utc(), clock);
}
