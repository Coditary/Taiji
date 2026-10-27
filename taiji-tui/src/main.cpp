#include "chat_app.hpp"

#include <cstdlib>
#include <cstring>
#include <tuinator/tuinator.hpp>

int main(int argc, char** argv) {
    bool local_mode = false;
    for (int i = 1; i < argc; ++i) {
        if (std::strcmp(argv[i], "--local") == 0 || std::strcmp(argv[i], "--local-only") == 0) {
            local_mode = true;
        }
    }
    if (local_mode) {
        setenv("TAIJI_LOCAL", "1", 1);
    }

    tuinator::Application app;
    ChatApp chat(app);
    chat.set_root();
    return app.run();
}
