#include <tuinator/testing/app_driver.hpp>
#include <tuinator/testing/json_protocol.hpp>

#include "scenes/scenes.hpp"

#include <cstdlib>
#include <iostream>
#include <memory>
#include <string>

int main(int argc, char** argv) {
    std::string scene = "composer";
    if (argc > 1 && argv[1] != nullptr && argv[1][0] != '\0') {
        scene = argv[1];
    }
    if (const char* env = std::getenv("TAIJI_TEST_SCENE"); env != nullptr && env[0] != '\0') {
        scene = env;
    }

    std::cerr << "taiji-tui-test-agent scene=" << scene << '\n';
    tuinator::testing::AppDriver driver({80, 24}, taiji::ui::test_scenes::build_scene_root(scene));
    driver.present();
    return tuinator::testing::run_json_protocol_loop(driver);
}
