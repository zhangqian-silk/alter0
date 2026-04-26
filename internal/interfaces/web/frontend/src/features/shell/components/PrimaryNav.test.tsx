import { fireEvent, render, screen } from "@testing-library/react";
import { PrimaryNav } from "./PrimaryNav";

describe("PrimaryNav", () => {
  it("renders text-only brand chrome and locale controls in the expanded sidebar", () => {
    const { container } = render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={vi.fn()}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={vi.fn()}
      />,
    );

    expect(container.querySelector(".brand-mark")).not.toBeInTheDocument();
    expect(screen.getByText("Alter0")).toBeInTheDocument();
    expect(container.querySelector(".nav-profile")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Language" })).toBeInTheDocument();
  });

  it("keeps route navigation and the collapse control interactive", () => {
    const onNavigate = vi.fn();
    const onToggleNavCollapsed = vi.fn();

    render(
      <PrimaryNav
        currentRoute="chat"
        language="en"
        navCollapsed={false}
        onNavigate={onNavigate}
        onToggleLanguage={vi.fn()}
        onToggleNavCollapsed={onToggleNavCollapsed}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Terminal" }));
    fireEvent.click(screen.getByRole("button", { name: "Collapse navigation" }));

    expect(onNavigate).toHaveBeenCalledWith("terminal");
    expect(onToggleNavCollapsed).toHaveBeenCalledTimes(1);
  });
});
