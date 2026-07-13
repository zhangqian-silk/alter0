import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MessageMarkdownShell } from "./MessageMarkdownShell";

describe("MessageMarkdownShell", () => {
  it("renders markdown through the shared shell (toolbar removed — copy button lives in msg-meta footer)", () => {
    const { container } = render(
      <MessageMarkdownShell
        markdown={"# Message\n\nUse `pwd` before opening [Chat](/chat)."}
        className="chatRuntime-final-text"
        bodyClassName="chatRuntime-final-rendered"
      />,
    );

    const shell = container.querySelector(".message-markdown-shell");
    // Only one child: the body. No toolbar div anymore.
    expect(shell?.children.length).toBe(1);
    expect(shell?.children.item(0)).toHaveClass("message-markdown-body");
    expect(container.querySelector(".chatRuntime-final-rendered")).toContainHTML("<h1>Message</h1>");
    expect(container.querySelector(".chat-md-inline-code")).toHaveTextContent("pwd");
    expect(screen.getByRole("link", { name: "Chat" })).toHaveAttribute("href", "/chat");
    // Toolbar is gone from this component
    expect(container.querySelector(".message-markdown-toolbar")).not.toBeInTheDocument();
  });

  it("keeps rendered text nodes stable across unrelated parent renders", () => {
    function Harness() {
      const [count, setCount] = useState(0);
      return (
        <>
          <button type="button" onClick={() => setCount((current) => current + 1)}>
            rerender {count}
          </button>
          <MessageMarkdownShell
            markdown="Hello. What should we work on?"
          />
        </>
      );
    }

    const { container } = render(<Harness />);
    const paragraph = container.querySelector(".message-markdown-rendered p");

    fireEvent.click(screen.getByRole("button", { name: /rerender/i }));

    expect(container.querySelector(".message-markdown-rendered p")).toBe(paragraph);
  });
});
