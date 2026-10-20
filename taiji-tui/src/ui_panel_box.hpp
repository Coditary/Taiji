#pragma once

#include <tuinator/layout/box.hpp>
#include <tuinator/render/paint_context.hpp>

/// VBox that fills its bounds with the main chat panel color before painting children.
class MainPanelVBox : public tuinator::VBox {
  public:
    explicit MainPanelVBox(tuinator::BoxOptions options = {});

    void paint(tuinator::PaintContext& ctx) const override;
};

/// VBox that fills its bounds with the sidebar panel color before painting children.
class SidebarPanelVBox : public tuinator::VBox {
  public:
    explicit SidebarPanelVBox(tuinator::BoxOptions options = {});

    void paint(tuinator::PaintContext& ctx) const override;
};
