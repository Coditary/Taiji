#include "chat_app.hpp"

#include <tuinator/tuinator.hpp>

int main() {
    tuinator::Application app;
    ChatApp chat(app);
    chat.set_root();
    return app.run();
}
