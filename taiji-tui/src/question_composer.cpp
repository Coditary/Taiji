#include "question_composer.hpp"

#include "ui_locale.hpp"

#include <tuinator/render/text.hpp>

#include <algorithm>
#include <cctype>
#include <sstream>

namespace {

std::string trim_copy(std::string_view text) {
    const std::size_t start = text.find_first_not_of(" \t\r\n");
    if (start == std::string_view::npos) {
        return {};
    }
    const std::size_t end = text.find_last_not_of(" \t\r\n");
    return std::string(text.substr(start, end - start + 1));
}

std::string json_extract_string(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return {};
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size() || json[i] != '"') {
        return {};
    }
    ++i;
    std::string out;
    while (i < json.size()) {
        const char ch = json[i++];
        if (ch == '\\' && i < json.size()) {
            const char esc = json[i++];
            switch (esc) {
            case '"':
            case '\\':
            case '/':
                out.push_back(esc);
                break;
            case 'b':
                out.push_back('\b');
                break;
            case 'f':
                out.push_back('\f');
                break;
            case 'n':
                out.push_back('\n');
                break;
            case 'r':
                out.push_back('\r');
                break;
            case 't':
                out.push_back('\t');
                break;
            default:
                out.push_back(esc);
                break;
            }
            continue;
        }
        if (ch == '"') {
            break;
        }
        out.push_back(ch);
    }
    return out;
}

std::string json_extract_value(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return {};
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    if (i >= json.size()) {
        return {};
    }
    if (json[i] == '"') {
        return json_extract_string(json, key);
    }
    if (json[i] == '{' || json[i] == '[') {
        const char open = json[i];
        const char close = open == '{' ? '}' : ']';
        int depth = 0;
        const std::size_t start = i;
        for (; i < json.size(); ++i) {
            const char ch = json[i];
            if (ch == '"') {
                ++i;
                while (i < json.size()) {
                    if (json[i] == '\\' && i + 1 < json.size()) {
                        i += 2;
                        continue;
                    }
                    if (json[i] == '"') {
                        break;
                    }
                    ++i;
                }
                continue;
            }
            if (ch == open) {
                ++depth;
            } else if (ch == close) {
                --depth;
                if (depth == 0) {
                    return json.substr(start, i - start + 1);
                }
            }
        }
        return {};
    }
    const std::size_t start = i;
    while (i < json.size() && json[i] != ',' && json[i] != '}' && json[i] != ']') {
        ++i;
    }
    return trim_copy(json.substr(start, i - start));
}

bool json_extract_bool(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    const std::size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return false;
    }
    std::size_t i = pos + needle.size();
    while (i < json.size() && (json[i] == ' ' || json[i] == '\t')) {
        ++i;
    }
    return json.compare(i, 4, "true") == 0;
}

std::vector<std::string> json_extract_object_array(const std::string& json, const std::string& key) {
    const std::string array_json = json_extract_value(json, key);
    if (array_json.empty() || array_json.front() != '[') {
        return {};
    }

    std::vector<std::string> out;
    std::size_t i = 1;
    while (i < array_json.size()) {
        while (i < array_json.size() && (array_json[i] == ' ' || array_json[i] == '\t' || array_json[i] == ',')) {
            ++i;
        }
        if (i >= array_json.size() || array_json[i] == ']') {
            break;
        }
        if (array_json[i] != '{') {
            break;
        }
        int depth = 0;
        const std::size_t start = i;
        for (; i < array_json.size(); ++i) {
            const char ch = array_json[i];
            if (ch == '"') {
                ++i;
                while (i < array_json.size()) {
                    if (array_json[i] == '\\' && i + 1 < array_json.size()) {
                        i += 2;
                        continue;
                    }
                    if (array_json[i] == '"') {
                        break;
                    }
                    ++i;
                }
                continue;
            }
            if (ch == '{') {
                ++depth;
            } else if (ch == '}') {
                --depth;
                if (depth == 0) {
                    out.push_back(array_json.substr(start, i - start + 1));
                    ++i;
                    break;
                }
            }
        }
    }
    return out;
}

std::string json_escape(const std::string& text) {
    std::string out;
    out.reserve(text.size() + 8);
    for (const char ch : text) {
        switch (ch) {
        case '"':
            out += "\\\"";
            break;
        case '\\':
            out += "\\\\";
            break;
        case '\n':
            out += "\\n";
            break;
        case '\r':
            out += "\\r";
            break;
        case '\t':
            out += "\\t";
            break;
        default:
            out.push_back(ch);
            break;
        }
    }
    return out;
}

QuestionComposer::Option parse_option(const std::string& object_json) {
    QuestionComposer::Option opt;
    opt.label = json_extract_string(object_json, "label");
    opt.description = json_extract_string(object_json, "description");
    return opt;
}

QuestionComposer::QuestionInfo parse_question_info(const std::string& object_json) {
    QuestionComposer::QuestionInfo info;
    info.id = json_extract_string(object_json, "id");
    info.header = json_extract_string(object_json, "header");
    info.question = json_extract_string(object_json, "question");
    info.multiple = json_extract_bool(object_json, "multiple");
    if (object_json.find("\"custom\"") != std::string::npos) {
        info.custom = json_extract_bool(object_json, "custom");
    }
    for (const std::string& opt_json : json_extract_object_array(object_json, "options")) {
        QuestionComposer::Option opt = parse_option(opt_json);
        if (!opt.label.empty()) {
            info.options.push_back(std::move(opt));
        }
    }
    if (info.header.empty() && !info.question.empty()) {
        info.header = info.question.size() > 30 ? info.question.substr(0, 27) + "..." : info.question;
    }
    return info;
}

int clamp_width(int width) { return std::max(16, width); }

std::string truncate_text(const std::string& text, int max_cols) {
    if (max_cols <= 0) {
        return {};
    }
    if (static_cast<int>(text.size()) <= max_cols) {
        return text;
    }
    if (max_cols <= 3) {
        return text.substr(0, static_cast<std::size_t>(max_cols));
    }
    return text.substr(0, static_cast<std::size_t>(max_cols - 3)) + "...";
}

} // namespace

bool parse_user_prompt_event(const std::string& json, std::string& request_id,
                             std::vector<QuestionComposer::QuestionInfo>& questions) {
    if (json_extract_string(json, "type") != "user_prompt") {
        return false;
    }
    const std::string data_json = json_extract_value(json, "data");
    if (data_json.empty()) {
        return false;
    }
    request_id = json_extract_string(data_json, "request_id");
    questions.clear();
    for (const std::string& item : json_extract_object_array(data_json, "questions")) {
        QuestionComposer::QuestionInfo info = parse_question_info(item);
        if (!info.question.empty() && info.options.size() >= 2) {
            questions.push_back(std::move(info));
        }
    }
    return !request_id.empty() && !questions.empty();
}

std::string encode_question_answers_json(const std::vector<std::vector<std::string>>& answers) {
    std::ostringstream out;
    out << '[';
    for (std::size_t i = 0; i < answers.size(); ++i) {
        if (i > 0) {
            out << ',';
        }
        out << '[';
        for (std::size_t j = 0; j < answers[i].size(); ++j) {
            if (j > 0) {
                out << ',';
            }
            out << '"' << json_escape(answers[i][j]) << '"';
        }
        out << ']';
    }
    out << ']';
    return out.str();
}

QuestionComposer::QuestionComposer(tuinator::Style style, tuinator::Style focused_style)
    : style_(std::move(style)), focused_style_(std::move(focused_style)) {}

void QuestionComposer::set_wrap_width(int width) { wrap_width_ = clamp_width(width); }

void QuestionComposer::set_on_submit(
    std::function<void(std::string request_id, std::vector<std::vector<std::string>> answers)> callback) {
    on_submit_ = std::move(callback);
}

void QuestionComposer::set_on_reject(std::function<void(std::string request_id)> callback) {
    on_reject_ = std::move(callback);
}

void QuestionComposer::begin(std::string request_id, std::vector<QuestionInfo> questions) {
    request_id_ = std::move(request_id);
    questions_ = std::move(questions);
    answers_.assign(questions_.size(), {});
    multi_selected_.clear();
    current_tab_ = 0;
    selected_row_ = 0;
    editing_custom_ = false;
    custom_input_.reset();
    active_ = !questions_.empty();
    rebuild_rows();
    mark_layout_dirty();
    mark_dirty();
}

void QuestionComposer::clear() {
    request_id_.clear();
    questions_.clear();
    answers_.clear();
    multi_selected_.clear();
    visible_rows_.clear();
    current_tab_ = 0;
    selected_row_ = 0;
    editing_custom_ = false;
    custom_input_.reset();
    active_ = false;
    mark_layout_dirty();
    mark_dirty();
}

const QuestionComposer::QuestionInfo& QuestionComposer::current_question() const {
    return questions_.at(static_cast<std::size_t>(current_tab_));
}

int QuestionComposer::option_count() const {
    if (!active_) {
        return 0;
    }
    return static_cast<int>(current_question().options.size());
}

int QuestionComposer::custom_index() const {
    if (!active_ || !current_question().custom) {
        return -1;
    }
    return option_count();
}

void QuestionComposer::rebuild_rows() {
    visible_rows_.clear();
    if (!active_) {
        return;
    }
    const QuestionInfo& q = current_question();
    for (int i = 0; i < static_cast<int>(q.options.size()); ++i) {
        visible_rows_.push_back(VisibleRow{.option_index = i, .custom = false});
    }
    if (q.custom) {
        visible_rows_.push_back(VisibleRow{.option_index = -1, .custom = true});
    }
    selected_row_ = std::clamp(selected_row_, 0, std::max(0, static_cast<int>(visible_rows_.size()) - 1));
}

int QuestionComposer::display_row_count() const { return static_cast<int>(visible_rows_.size()); }

QuestionComposer::VisibleRow QuestionComposer::row_at(int display_row) const {
    if (display_row < 0 || display_row >= display_row_count()) {
        return {};
    }
    return visible_rows_.at(static_cast<std::size_t>(display_row));
}

void QuestionComposer::set_selected_row(int row) {
    if (row < 0 || row >= display_row_count()) {
        return;
    }
    if (row == selected_row_) {
        return;
    }
    selected_row_ = row;
    mark_dirty();
}

void QuestionComposer::toggle_multi_selection(int option_index) {
    if (option_index < 0 || option_index >= option_count()) {
        return;
    }
    const auto it = std::find(multi_selected_.begin(), multi_selected_.end(), option_index);
    if (it == multi_selected_.end()) {
        multi_selected_.push_back(option_index);
    } else {
        multi_selected_.erase(it);
    }
    mark_dirty();
}

void QuestionComposer::pick_single(int option_index, bool custom) {
    const QuestionInfo& q = current_question();
    std::vector<std::string> picked;
    if (custom) {
        if (custom_input_ != nullptr) {
            const std::string text = trim_copy(custom_input_->value());
            if (!text.empty()) {
                picked.push_back(text);
            }
        }
    } else if (option_index >= 0 && option_index < option_count()) {
        picked.push_back(q.options.at(static_cast<std::size_t>(option_index)).label);
    }
    if (picked.empty()) {
        return;
    }
    answers_.at(static_cast<std::size_t>(current_tab_)) = std::move(picked);
    advance_or_submit();
}

void QuestionComposer::advance_or_submit() {
    if (current_tab_ + 1 < static_cast<int>(questions_.size())) {
        current_tab_ += 1;
        multi_selected_.clear();
        selected_row_ = 0;
        editing_custom_ = false;
        custom_input_.reset();
        rebuild_rows();
        mark_layout_dirty();
        mark_dirty();
        return;
    }
    submit_all();
}

void QuestionComposer::submit_all() {
    if (!on_submit_ || request_id_.empty()) {
        clear();
        return;
    }
    const std::string request_id = request_id_;
    std::vector<std::vector<std::string>> answers = answers_;
    clear();
    on_submit_(request_id, std::move(answers));
}

void QuestionComposer::begin_custom_edit() {
    tuinator::TextInputOptions options;
    options.min_width = 16;
    options.chrome = tuinator::TextInputChrome::Framed;
    options.max_content_lines = 1;
    options.prompt = "› ";
    const std::string ph = ui_shell("question.custom_placeholder");
    options.placeholder = ph.empty() ? "Custom answer…" : ph;
    custom_input_ = std::make_unique<tuinator::TextInput>(options, style_, focused_style_);
    custom_input_->set_on_submit([this](const std::string&) { finish_custom_edit(true); });
    editing_custom_ = true;
    mark_layout_dirty();
    mark_dirty();
}

void QuestionComposer::finish_custom_edit(bool keep) {
    if (!editing_custom_) {
        return;
    }
    if (keep) {
        if (current_question().multiple) {
            const std::string text = custom_input_ != nullptr ? trim_copy(custom_input_->value()) : std::string{};
            if (!text.empty()) {
                answers_.at(static_cast<std::size_t>(current_tab_)).push_back(text);
            }
            editing_custom_ = false;
            custom_input_.reset();
            mark_layout_dirty();
            mark_dirty();
            return;
        }
        pick_single(-1, true);
        return;
    }
    editing_custom_ = false;
    custom_input_.reset();
    mark_layout_dirty();
    mark_dirty();
}

std::string QuestionComposer::footer_hint() const {
    if (editing_custom_) {
        const std::string hint = ui_shell("question.footer_apply");
        return hint.empty() ? "Enter: apply  ·  Esc: back" : hint;
    }
    if (current_question().multiple) {
        std::string hint = "↑↓ select  ·  Space: toggle  ·  Enter: next";
        if (questions_.size() > 1) {
            hint += "  ·  Tab: switch question";
        }
        return hint;
    }
    std::string hint = "↑↓ select  ·  Enter: confirm";
    if (questions_.size() > 1) {
        hint += "  ·  Tab: switch question";
    }
    if (current_question().custom) {
        hint += "  ·  custom answer available";
    }
    return hint;
}

tuinator::Size QuestionComposer::preferred_size() const {
    if (!active_) {
        return {0, 0};
    }
    int height = 3 + display_row_count();
    if (editing_custom_) {
        height += 2;
    }
    return {clamp_width(wrap_width_), height};
}

void QuestionComposer::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    if (editing_custom_ && custom_input_ != nullptr) {
        const int input_height = 1;
        const int y = std::max(0, bounds.height - input_height - 1);
        custom_input_->layout({0, y, bounds.width, input_height});
    }
}

void QuestionComposer::paint(tuinator::PaintContext& ctx) const {
    if (!active_) {
        return;
    }

    const tuinator::Rect rect = bounds();
    if (rect.width <= 0 || rect.height <= 0) {
        return;
    }

    tuinator::Style frame = style_;
    frame.background_rgb = tuinator::rgb(42, 38, 58);
    frame.foreground_rgb = tuinator::rgb(230, 228, 240);

    tuinator::Style heading = frame;
    heading.foreground_rgb = tuinator::rgb(255, 255, 255);

    tuinator::Style hint = frame;
    hint.foreground_rgb = tuinator::rgb(140, 135, 160);
    hint.dim = true;

    tuinator::Canvas& canvas = ctx.canvas;
    canvas.fill_rect({0, 0, rect.width, rect.height}, ' ', frame);

    const QuestionInfo& q = current_question();
    std::ostringstream title;
    title << q.header;
    if (questions_.size() > 1) {
        title << "  (" << (current_tab_ + 1) << '/' << questions_.size() << ')';
    }
    canvas.draw_text({0, 0}, title.str(), heading);

    const int question_width = std::max(0, rect.width);
    canvas.draw_text({0, 1}, truncate_text(q.question, question_width), frame);

    int y = 2;
    for (int row = 0; row < display_row_count(); ++row) {
        const VisibleRow visible = row_at(row);
        const bool selected = row == selected_row_;
        tuinator::Style row_style = selected ? focused_style_ : frame;
        row_style.background_rgb = frame.background_rgb;

        std::string prefix;
        if (q.multiple && !visible.custom && visible.option_index >= 0) {
            const bool checked =
                std::find(multi_selected_.begin(), multi_selected_.end(), visible.option_index) != multi_selected_.end();
            prefix = checked ? "[x] " : "[ ] ";
        } else {
            prefix = selected ? "› " : "  ";
        }

        std::string label;
        if (visible.custom) {
            {
                const std::string custom = ui_shell("question.custom_label");
                label = custom.empty() ? "Custom answer" : custom;
            }
        } else {
            label = q.options.at(static_cast<std::size_t>(visible.option_index)).label;
        }

        canvas.draw_text({0, y}, prefix + label, row_style);
        ++y;

        if (!visible.custom) {
            const std::string& desc = q.options.at(static_cast<std::size_t>(visible.option_index)).description;
            if (!desc.empty() && y < rect.height) {
                canvas.draw_text({2, y}, desc, hint);
                ++y;
            }
        }
    }

    if (y < rect.height) {
        canvas.draw_text({0, rect.height - 1}, footer_hint(), hint);
    }

    if (editing_custom_ && custom_input_ != nullptr) {
        custom_input_->paint(ctx);
    }
}

tuinator::Widget* QuestionComposer::hit_test(tuinator::Point point) {
    if (!active_ || !bounds_.contains(point)) {
        return nullptr;
    }
    if (editing_custom_ && custom_input_ != nullptr) {
        if (tuinator::Widget* hit = custom_input_->hit_test(point)) {
            return hit;
        }
    }
    return this;
}

bool QuestionComposer::handle_event(const tuinator::Event& event) {
    if (!active_) {
        return false;
    }

    if (editing_custom_ && custom_input_ != nullptr) {
        if (const auto* key = std::get_if<tuinator::KeyPress>(&event)) {
            if (key->key == tuinator::Key::Escape) {
                finish_custom_edit(false);
                return true;
            }
        }
        return custom_input_->handle_event(event);
    }

    const auto* key = std::get_if<tuinator::KeyPress>(&event);
    if (!key) {
        return true;
    }

    if (key->key == tuinator::Key::Escape) {
        if (!on_reject_ || request_id_.empty()) {
            clear();
            return true;
        }
        const std::string request_id = request_id_;
        clear();
        on_reject_(request_id);
        return true;
    }

    if (key->key == tuinator::Key::Tab) {
        if (questions_.size() > 1) {
            if (key->shift) {
                current_tab_ = (current_tab_ - 1 + static_cast<int>(questions_.size())) % static_cast<int>(questions_.size());
            } else {
                current_tab_ = (current_tab_ + 1) % static_cast<int>(questions_.size());
            }
            multi_selected_.clear();
            selected_row_ = 0;
            rebuild_rows();
            mark_layout_dirty();
            mark_dirty();
        }
        return true;
    }

    if (key->key == tuinator::Key::Up) {
        if (display_row_count() > 0) {
            set_selected_row((selected_row_ - 1 + display_row_count()) % display_row_count());
        }
        return true;
    }
    if (key->key == tuinator::Key::Down) {
        if (display_row_count() > 0) {
            set_selected_row((selected_row_ + 1) % display_row_count());
        }
        return true;
    }

    const VisibleRow selected = row_at(selected_row_);
    if (key->character == ' ' && current_question().multiple && selected.option_index >= 0) {
        toggle_multi_selection(selected.option_index);
        return true;
    }

    if (key->key == tuinator::Key::Enter) {
        if (selected.custom) {
            begin_custom_edit();
            return true;
        }
        if (current_question().multiple) {
            std::vector<std::string> picked;
            for (const int idx : multi_selected_) {
                if (idx >= 0 && idx < option_count()) {
                    picked.push_back(current_question().options.at(static_cast<std::size_t>(idx)).label);
                }
            }
            if (picked.empty()) {
                return true;
            }
            answers_.at(static_cast<std::size_t>(current_tab_)) = std::move(picked);
            advance_or_submit();
            return true;
        }
        pick_single(selected.option_index, false);
        return true;
    }

    return true;
}

ComposerSwitch::ComposerSwitch(std::unique_ptr<tuinator::Widget> input, std::unique_ptr<tuinator::Widget> question) {
    input_ = input.get();
    question_ = question.get();
    add_child(std::move(input));
    add_child(std::move(question));
}

void ComposerSwitch::show_input() {
    showing_question_ = false;
    mark_layout_dirty();
    mark_dirty();
}

void ComposerSwitch::show_question() {
    showing_question_ = true;
    mark_layout_dirty();
    mark_dirty();
}

tuinator::Widget* ComposerSwitch::active_child() const {
    if (showing_question_) {
        return question_;
    }
    return input_;
}

bool ComposerSwitch::has_focused_descendant() const {
    const tuinator::Widget* child = active_child();
    return child != nullptr && child->has_focused_descendant();
}

bool ComposerSwitch::wants_initial_focus() const {
    tuinator::Widget* child = active_child();
    return child != nullptr && child->wants_initial_focus();
}

tuinator::Size ComposerSwitch::preferred_size() const {
    tuinator::Widget* child = active_child();
    if (child == nullptr) {
        return {0, 0};
    }
    return child->preferred_size();
}

void ComposerSwitch::layout(tuinator::Rect bounds) {
    bounds_ = bounds;
    const tuinator::Rect hidden{bounds.x, bounds.y, 0, 0};
    if (input_ != nullptr) {
        input_->layout(showing_question_ ? hidden : bounds);
    }
    if (question_ != nullptr) {
        question_->layout(showing_question_ ? bounds : hidden);
    }
}

void ComposerSwitch::paint(tuinator::PaintContext& ctx) const {
    tuinator::Widget* child = active_child();
    if (child != nullptr) {
        child->paint(ctx);
    }
}

bool ComposerSwitch::handle_event(const tuinator::Event& event) {
    tuinator::Widget* child = active_child();
    if (child == nullptr) {
        return false;
    }
    return child->handle_event(event);
}

tuinator::Widget* ComposerSwitch::hit_test(tuinator::Point point) {
    tuinator::Widget* child = active_child();
    if (child == nullptr) {
        return nullptr;
    }
    return child->hit_test(point);
}

tuinator::Widget* ComposerSwitch::hit_test_focusable(tuinator::Point point) {
    if (!bounds_.contains(point)) {
        return nullptr;
    }
    tuinator::Widget* child = active_child();
    if (child == nullptr) {
        return nullptr;
    }
    return child->hit_test_focusable(point);
}

void ComposerSwitch::collect_focusable(std::vector<tuinator::Widget*>& out) {
    tuinator::Widget* child = active_child();
    if (child != nullptr) {
        child->collect_focusable(out);
    }
}
